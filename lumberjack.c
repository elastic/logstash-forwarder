#define _BSD_SOURCE /* to get gethostname() under linux/gcc */
#include <sys/types.h>
#include <sys/resource.h> /* for setrlimit */
#include <getopt.h>
#include "insist.h"
#include <pthread.h>
#include <unistd.h> /* for gethostname */
#include "zmq.h"
#include "harvester.h"
#include "emitter.h"
#include "jemalloc/jemalloc.h"
#include <signal.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include "proto.h"
#include "flog.h"

#define ZMQ_EMITTER_ENDPOINT "inproc://emitter"

typedef enum {
  opt_help = 'h',
  opt_version = 'v',
  opt_field,
  opt_ssl_ca_path,
  opt_host,
  opt_port,
  opt_window_size,
} optlist_t;

struct option_doc {
  const char *name;
  int         has_arg;
  int         val;
  const char *documentation;
};

static struct option_doc options[] = {
  { "help", no_argument, opt_help, "show this help" },
  { "version", no_argument, opt_version, "show the version of lumberjack" },

  /* Support arbitrary fields in the events, like:
   * ./lumberjack --field host=$(hostname) --field role=frontend .../access.log
   *
   * This will allow you to send any arbitrary data along with every event.
   * { "host": "foo", "file": "/path/to/file.log", "message": "hello ..." }
   */
  { "field", required_argument, opt_field, 
    "Add a custom key-value mapping to every line emitted" },

  /* ssl cert and key, optional */
  //{ "ssl-certificate", required_argument, NULL, opt_ssl_certificate },
  //{ "ssl-key", required_argument, NULL, opt_ssl_key },
  /* TODO(sissel): How to provide key passphrase/credentials? */

  /* What cert authority to trust. This can be the path to a single self-signed
   * certificate if you choose. */
  { "ssl-ca-path", required_argument, opt_ssl_ca_path, 
    "Set the trusted cert/ca path for lumberjack's ssl client. " \
    "Can be a file or a directory." },
  { "host", required_argument, opt_host,
    "The hostname to send lumberjack messages to. You can specify multiple " \
    "by separating hosts with a comma." },
  { "port", required_argument, opt_port,
    "The port to connect on the lumberjack server" },
  { "window-size", required_argument, opt_window_size,
    "The maximum number of outstanding messages to send before we will " \
    "wait for an acknowledgement" },
  { NULL, 0, 0, NULL },
};

void usage(const char *prog) {
  printf("Usage: %s [options] /path/to/file [/path/to/file2 ...]\n", prog);

  for (int i = 0; options[i].name != NULL; i++) {
    printf("  --%s%s %.*s %s\n", options[i].name,
           options[i].has_arg ? " VALUE" : "",
           (int)(20 - strlen(options[i].name) - (options[i].has_arg ? 6 : 0)),
           "                                   ",
           options[i].documentation);
  }
} /* usage */

void set_resource_limits(int file_count) {
  struct rlimit limits;
  int rc;

  rc = nice(1); /* ask for less priority in the scheduler */
  insist(rc != -1, "nice(1) failed: %s", strerror(errno));

  /* Only set resource limits if not running under valgrind.
   * If we set limits under valgrind, it crashes due to exceeding said limits
   */

  if ((getenv("LD_PRELOAD") != NULL) \
      && (strstr(getenv("LD_PRELOAD"), "/vgpreload_") != NULL)) {
    flog(stdout, "Valgrind detected, skipping self-resource limitations");
    return;
  }

  /* Set open file limit 
   * 3 'open files' per log file watched:
   *   - one for the file itself
   *   - two for the socketpair in zeromq
   * */
  limits.rlim_cur = limits.rlim_max = (file_count * 3 ) + 100;
  flog(stdout, "Watching %d files, setting open file limit to %ld",
         file_count, limits.rlim_max);
  rc = setrlimit(RLIMIT_NOFILE, &limits);
  insist(rc != -1, "setrlimit(RLIMIT_NOFILE, ... %d) failed: %s",
         (int)limits.rlim_max, strerror(errno));

  /* I'd like to set RLIMIT_NPROC, but that setting applies to the entire user
   * for all processes, not just subprocesses or threads belonging to this
   * process. */
  //limits.rlim_cur = limits.rlim_max = file_count + 10;
  //rc = setrlimit(RLIMIT_NPROC, &limits);
  //insist(rc != -1, "setrlimit(RLIMIT_NPROC, ... %d) failed: %s\n",
         //(int)limits.rlim_max, strerror(errno));

  /* Set resident memory limit */
  /* Allow 1mb per file opened */
  int bytes = (1<<20) * file_count;
  /* RLIMIT_RSS uses 'pages' as the unit, convert bytes to pages. */
  limits.rlim_cur = limits.rlim_max = (int)(bytes / sysconf(_SC_PAGESIZE));
  flog(stdout, "Watching %d files, setting memory usage limit to %d bytes",
       file_count, bytes); 
  rc = setrlimit(RLIMIT_RSS, &limits);
  insist(rc != -1, "setrlimit(RLIMIT_RSS, %d pages (%d bytes)) failed: %s",
         (int)limits.rlim_max, bytes, strerror(errno));
} /* set_resource_limits */

int main(int argc, char **argv) {
  int c, i;
  struct emitter_config emitter_config;
  struct option *getopt_options = NULL;

  struct kv *extra_fields = NULL;
  size_t extra_fields_len = 0;

  /* defaults */
  memset(&emitter_config, 0, sizeof(struct emitter_config));
  emitter_config.port = 5001;
  emitter_config.window_size = 4096;
  
  /* convert the 'option_doc' array into a 'struct option' array 
   * for use with getopt_long_only */
  for (i = 0; options[i].name != NULL; i++) {
    getopt_options = realloc(getopt_options, (i+1) * sizeof(struct option));
    getopt_options[i].name = options[i].name;
    getopt_options[i].has_arg = options[i].has_arg;
    getopt_options[i].flag = NULL;
    getopt_options[i].val = options[i].val;
  }

  /* Add one last item for the list terminator NULL */
  getopt_options = realloc(getopt_options, (i+1) * sizeof(struct option));
  getopt_options[i].name = NULL;

  char *tmp;
  while (i = -1, c = getopt_long_only(argc, argv, "+hv", getopt_options, &i), c != -1) {
    /* TODO(sissel): handle args */
    switch (c) {
      case opt_ssl_ca_path:
        emitter_config.ssl_ca_path = strdup(optarg);
        break;
      case opt_version:
        printf("version unknown. Could be awesome.\n");
        break;
      case opt_help:
        usage(argv[0]);
        return 0;
      case opt_host:
        emitter_config.host = strdup(optarg);
        break;
      case opt_port:
        emitter_config.port = (unsigned short)atoi(optarg);
        break;
      case opt_window_size:
        emitter_config.window_size = (size_t)atoi(optarg);
        printf("size: %d\n", (int)emitter_config.window_size);
        break;
      case opt_field:
        tmp = strchr(optarg, '=');
        if (tmp == NULL) {
          printf("Invalid --field setting, expected 'foo=bar' form, " \
                 "didn't see '=' in '%s'", optarg);
          usage(argv[0]);
          exit(1);
        }
        extra_fields_len += 1;
        extra_fields = realloc(extra_fields, extra_fields_len * sizeof(struct kv));
        *tmp = '\0'; // turn '=' into null terminator
        tmp++; /* skip to first char of value */
        extra_fields[extra_fields_len - 1].key = strdup(optarg);
        extra_fields[extra_fields_len - 1].key_len = strlen(optarg);
        extra_fields[extra_fields_len - 1].value = strdup(tmp);
        extra_fields[extra_fields_len - 1].value_len = strlen(tmp);
        break;
      default:
        insist(i == -1, "Flag (--%s%s%s) known, but someone forgot to " \
               "implement handling of it! This is certainly a bug.",
               options[i].name, 
               options[i].has_arg ? " " : "",
               options[i].has_arg ? optarg : "");

        usage(argv[0]);
        return 1;
    }
  }
  free(getopt_options);

  if (emitter_config.host == NULL) {
    printf("Missing --host flag\n");
    usage(argv[0]);
    return 1;
  }

  if (emitter_config.port == 0) {
    printf("Missing --port flag\n");
    usage(argv[0]);
    return 1;
  }

  argc -= optind;
  argv += optind;

  /* I'll handle write failures; no signals please */
  signal(SIGPIPE, SIG_IGN);

  insist(argc > 0, "No arguments given. What log files do you want shipped?");

  /* Set resource (memory, open file, etc) limits based on the
   * number of files being watched. */
  set_resource_limits(argc);

  pthread_t *harvesters = calloc(argc, sizeof(pthread_t));
  /* no I/O threads needed since we use inproc:// only */
  void *zmq = zmq_init(0 /* IO threads */); 

  /* Start harvesters for each path given */
  for (i = 0; i < argc; i++) {
    struct harvest_config *harvester = calloc(1, sizeof(struct harvest_config));
    harvester->zmq = zmq;
    harvester->zmq_endpoint = ZMQ_EMITTER_ENDPOINT;
    harvester->path = argv[i];
    harvester->fields = extra_fields;
    harvester->fields_len = extra_fields_len;
    pthread_create(&harvesters[i], NULL, harvest, harvester);
  }

  pthread_t emitter_thread;
  emitter_config.zmq = zmq;
  emitter_config.zmq_endpoint = ZMQ_EMITTER_ENDPOINT;
  pthread_create(&emitter_thread, NULL, emitter, &emitter_config);

  for (i = 0; i < argc; i++) {
    pthread_join(harvesters[i], NULL);
  }

  flog(stdout, "All harvesters completed. Exiting.");
  free(harvesters);

  /* TODO(sissel): Tell emitter to flush and exit */
  return 1;
} /* main */

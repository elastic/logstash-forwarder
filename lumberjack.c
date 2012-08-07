#define _BSD_SOURCE /* to get gethostname() under linux/gcc */
#include <sys/types.h>
#include <getopt.h>
#include <insist.h>
#include <pthread.h>
#include <unistd.h> /* for gethostname */
#include <zmq.h>
#include "harvester.h"
#include "emitter.h"
#include <jemalloc/jemalloc.h>
#include <signal.h>
#include <string.h>

#define ZMQ_EMITTER_ENDPOINT "inproc://emitter"

typedef enum {
  opt_help = 'h',
  opt_version = 'v',
  opt_field,
  opt_ssl_ca_path,
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

int main(int argc, char **argv) {
  int c, i;
  struct emitter_config emitter_config;
  struct option *getopt_options = NULL;
  
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

  argc -= optind;
  argv += optind;

  /* I'll handle write failures; no signals please */
  signal(SIGPIPE, SIG_IGN);

  insist(argc > 0, "No arguments given. What log files do you want shipped?");

  pthread_t *harvesters = calloc(argc, sizeof(pthread_t));
  /* no I/O threads needed since we use inproc:// only */
  void *zmq = zmq_init(0 /* IO threads */); 

  /* Start harvesters for each path given */
  for (i = 0; i < argc; i++) {
    struct harvest_config *harvester = calloc(1, sizeof(struct harvest_config));
    harvester->zmq = zmq;
    harvester->zmq_endpoint = ZMQ_EMITTER_ENDPOINT;
    harvester->path = argv[i];
    pthread_create(&harvesters[i], NULL, harvest, harvester);
  }

  emitter_config.zmq = zmq;
  emitter_config.zmq_endpoint = ZMQ_EMITTER_ENDPOINT;
  emitter(&emitter_config);

  /* Wait for the harvesters to die */
  for (i = 0; i < argc; i++) {
    pthread_join(harvesters[i], NULL);
  }
  exit(0);

  return 0;
} /* main */

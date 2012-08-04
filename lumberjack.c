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

typedef enum {
  opt_help = 'h',
  opt_version = 'v',
  opt_field,
} optlist_t;

static struct option options[] = {
  { "help", no_argument, NULL, opt_help },
  { "version", no_argument, NULL, opt_version },

  /* Support arbitrary fields in the events, like:
   * ./lumberjack --field host=$(hostname) --field role=frontend .../access.log
   *
   * This will allow you to send any arbitrary data along with every event.
   * { "host": "foo", "file": "/path/to/file.log", "message": "hello ..." }
   */
  { "field", required_argument, NULL, opt_field },
  { NULL, 0, NULL, 0 }
};

#define ZMQ_EMITTER_ENDPOINT "inproc://emitter"

int main(int argc, char **argv) {
  int c, i;
  while (c = getopt_long_only(argc, argv, "+hv", options, &i), c != -1) {
    /* handle args */
  }

  argc -= optind;
  argv += optind;

  insist(argc > 0, "No arguments given. What log files do you want shipped?");

  pthread_t *harvesters = calloc(argc, sizeof(pthread_t));
  /* no I/O threads needed since we use inproc:// only */
  void *zmq = zmq_init(0 /* IO threads */); 

  /* Start harvesters for each path given */
  for (i = 0; i < argc; i++) {
    struct harvest_config *config = calloc(1, sizeof(struct harvest_config));
    config->zmq = zmq;
    config->zmq_endpoint = ZMQ_EMITTER_ENDPOINT;
    config->path = argv[i];
    pthread_create(&harvesters[i], NULL, harvest, config);
  }

  struct emitter_config config;
  config.zmq = zmq;
  config.zmq_endpoint = ZMQ_EMITTER_ENDPOINT;
  emitter(&config);

  /* Wait for the harvesters to die */
  for (i = 0; i < argc; i++) {
    pthread_join(harvesters[i], NULL);
  }
  exit(0);

  return 0;
} /* main */

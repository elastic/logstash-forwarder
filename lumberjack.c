#define _BSD_SOURCE /* to get gethostname() under linux/gcc */
#include <sys/types.h>
#include <getopt.h>
#include <insist.h>
#include <pthread.h>
#include <unistd.h> /* for gethostname */
#include "harvester.h"

typedef enum {
  opt_help = 'h',
  opt_version = 'v',
} optlist_t;

static struct option options[] = {
  { "help", no_argument, NULL, opt_help },
  { "version", no_argument, NULL, opt_version },
  { NULL, 0, NULL, 0 }
};

static char hostname[200];

int main(int argc, char **argv) {
  int c, i;
  while (c = getopt_long_only(argc, argv, "+hv", options, &i), c != -1) {
    /* handle args */
  }

  argc -= optind;
  argv += optind;

  insist(argc > 0, "No arguments given. What log files do you want shipped?");

  gethostname(hostname, sizeof(hostname));

  pthread_t *harvesters = calloc(argc, sizeof(pthread_t));

  /* Start harvesters for each path given */
  for (i = 0; i < argc; i++) {
    pthread_create(&harvesters[i], NULL, harvest, argv[i]);
  }

  /* Wait for the harvesters to die */
  for (i = 0; i < argc; i++) {
    pthread_join(harvesters[i], NULL);
  }

  return 0;
} /* main */

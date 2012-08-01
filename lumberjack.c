#include <stdio.h>
#include <getopt.h>
#include <insist.h>

#include <pthread.h>
#include <unistd.h> /* for gethostname */

#include <string.h> /* for strerror(3) */
#include <errno.h> /* for errno */
#include <fcntl.h> /* for open(2) */

#include "backoff.h"

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

void *harvest(void *arg) {
  const char *path = (const char *)arg;
  int fd;
  fd = open(path, O_RDONLY);
  insist(fd >= 0, "open(%s) failed: %s", path, strerror(errno));

  char *buf;
  ssize_t bytes;
  buf = calloc(65536, sizeof(char));

  struct backoff sleeper;
  backoff_init(&sleeper, 10000 /* 10ms */, 15000000  /* 15 seconds */);

  for (;;) {
    bytes = read(fd, buf, 65536);
    if (bytes < 0) {
      /* error */
      break;
    } else if (bytes == 0) {
      backoff(&sleeper);
    } else {
      backoff_clear(&sleeper);
      printf("got: %.*s\n", (int)bytes, buf);

      /* Find newlines, emit an event */
      /* Event contents:
       *  - hostname
       *  - file
       *  - message
       */
      /* keep remainder in the buffer */
    }
  }
  close(fd);

  return NULL;
} /* harvest */

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
  for (int i = 0; i < argc; i++) {
    pthread_create(&harvesters[i], NULL, harvest, argv[i]);
  }

  /* Wait for the harvesters to die */
  for (int i = 0; i < argc; i++) {
    pthread_join(harvesters[i], NULL);
  }

  return 0;
} /* main */

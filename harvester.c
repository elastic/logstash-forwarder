#define _XOPEN_SOURCE 500 /* for useconds_t */
#include "harvester.h"
#include <string.h> /* for strerror(3) */
#include <errno.h> /* for errno */
#include <fcntl.h> /* for open(2) */
#include "backoff.h"
#include <unistd.h> /* for close, etc */
#include <stdio.h> /* printf and friends */

#include "insist.h"

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


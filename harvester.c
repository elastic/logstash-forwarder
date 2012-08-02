#define _BSD_SOURCE
#include <string.h> /* for strsep, etc */
#include "harvester.h"
#include <string.h> /* for strerror(3) */
#include <errno.h> /* for errno */
#include <fcntl.h> /* for open(2) */
#include "backoff.h"
#include <unistd.h> /* for close, etc */
#include <stdio.h> /* printf and friends */
#include <zmq.h>

#include "insist.h"

#define EMITTER_SOCKET "inproc://emitter"
#define BUFFERSIZE 16384

void *harvest(void *arg) {
  struct harvest_config *config = arg;
  int fd;
  int rc;

  fd = open(config->path, O_RDONLY);
  insist(fd >= 0, "open(%s) failed: %s", config->path, strerror(errno));

  char *buf;
  ssize_t bytes;
  buf = calloc(BUFFERSIZE, sizeof(char));

  struct backoff sleeper;
  backoff_init(&sleeper, 10000 /* 10ms */, 15000000  /* 15 seconds */);

  void *socket = zmq_socket(config->zmq, ZMQ_PUSH);
  insist(socket != NULL, "zmq_socket() failed: %s", strerror(errno));

  for (;;) {
    rc = zmq_connect(socket, config->zmq_endpoint);
    if (rc != 0 && errno == ECONNREFUSED) {
      backoff(&sleeper);
      continue; /* retry */
    } 
    insist(rc == 0, "zmq_connect(%s) failed: %s", config->zmq_endpoint,
           zmq_strerror(errno));
    break;
  }
  backoff_clear(&sleeper);

  int offset = 0;
  for (;;) {
    /* TODO(sissel): is truncation handled? */
    /* TODO(sissel): what about log rotation? */
    bytes = read(fd, buf + offset, BUFFERSIZE - offset);
    if (bytes < 0) {
      /* error */
      break;
    } else if (bytes == 0) {
      backoff(&sleeper);
    } else {
      backoff_clear(&sleeper);

      /* For each line, emit. Save the remainder */
      char *line;
      char *septok = buf;
      char *start = NULL;
      while (start = septok, (line = strsep(&septok, "\n")) != NULL) {
        if (septok == NULL) {
          /* last token found, no terminator though */
          offset = start - line;
          strcpy(buf + offset, buf);
        } else {
          /* emit line as an event */
          zmq_msg_t message;
          zmq_msg_init_data(&message, line, septok - start - 1, NULL, NULL);
          rc = zmq_send(socket, &message, 0);
          insist(rc == 0, "zmq_send() failed: %s", zmq_strerror(rc));
          zmq_msg_close(&message);
        }

      };

      /* Find newlines, emit an event */
      /* Event contents:
       *  - file
       *  - message
       *  - any arbitrary data the user wants
       *
       * Pick a serialization? msgpack?
       * host+file+message
       */
    }
  } /* loop forever, reading from a file */

  free(arg); /* allocated by the main method, up to us to free */
  close(fd);

  return NULL;
} /* harvest */


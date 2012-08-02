#include <string.h> /* for strsep, etc */
#include <string.h> /* for strerror(3) */
#include <errno.h> /* for errno */
#include <fcntl.h> /* for open(2) */
#include <unistd.h> /* for close, etc */
#include <stdio.h> /* printf and friends */
#include <zmq.h> /* zeromq messaging library */
#include <jansson.h> /* jansson JSON library */

#include "harvester.h"
#include "backoff.h"
#include "insist.h"

extern const char * HOSTNAME; /* lumberjack.c */
#define EMITTER_SOCKET "inproc://emitter"
#define BUFFERSIZE 16384

static struct timespec min_sleep = { 0, 10000000 }; /* 10ms */
static struct timespec max_sleep = { 15, 0 }; /* 15 */

void *harvest(void *arg) {
  struct harvest_config *config = arg;
  int fd;
  int rc;
  
  /* Make this so we only call it once. */
  char hostname[200];
  gethostname(hostname, sizeof(hostname));

  fd = open(config->path, O_RDONLY);
  insist(fd >= 0, "open(%s) failed: %s", config->path, strerror(errno));

  char *buf;
  ssize_t bytes;
  buf = calloc(BUFFERSIZE, sizeof(char));

  struct backoff sleeper;
  backoff_init(&sleeper, &min_sleep, &max_sleep);

  void *socket = zmq_socket(config->zmq, ZMQ_PUSH);
  insist(socket != NULL, "zmq_socket() failed: %s", strerror(errno));

  json_t *event = json_object();
  /* HOSTNAME is set globally by lumberjack.c */
  json_object_set_new(event, "host", json_string(hostname));
  json_object_set_new(event, "file", json_string(config->path));

  /* Wait for the zmq endpoint to be up (wait for connect to succeed) */
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
          char *serialized;
          json_t *line_obj = json_string(line);
          /* TODO(sissel): skip if line_obj is null (means line was invalid UTF-8) */

          json_object_set(event, "line", line_obj);
          /* TODO(sissel): include file offset */

          /* serialize to json */
          serialized = json_dumps(event, 0);

          /* Purge the 'line' from the event object so it'll be freed */
          json_object_del(event, "line");
          json_decref(line_obj);

          /* Ship it off to zeromq */
          //zmq_msg_init_data(&message, line, septok - start - 1, NULL, NULL);
          zmq_msg_init_data(&message, serialized, strlen(serialized) + 1, NULL, NULL);
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


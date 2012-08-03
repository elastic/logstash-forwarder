#define _BSD_SOURCE
#include <string.h> /* for strsep, strerror, etc */
#include <errno.h> /* for errno */
#include <fcntl.h> /* for open(2) */
#include <unistd.h> /* for close, etc */
#include <arpa/inet.h> /* for ntohl */
#include <stdio.h> /* printf and friends */
#include <zmq.h> /* zeromq messaging library */
#include <jansson.h> /* jansson JSON library */

#include "harvester.h"
#include "backoff.h"
#include "insist.h"

#define EMITTER_SOCKET "inproc://emitter"
#define BUFFERSIZE 16384

static struct timespec min_sleep = { 0, 10000000 }; /* 10ms */
static struct timespec max_sleep = { 15, 0 }; /* 15 */

/* A free function that simply calls free(3) for zmq_msg */
static inline void free2(void *data, void __attribute__((__unused__)) *hint) {
  free(data);
} /* free2 */

void *harvest(void *arg) {
  struct harvest_config *config = arg;
  int fd;
  int rc;
  char hostname[200];
  size_t hostname_len, path_len;
  
  /* Make this so we only call it once. */
  gethostname(hostname, sizeof(hostname));
  hostname_len = strlen(hostname);

  fd = open(config->path, O_RDONLY);
  path_len = strlen(config->path);
  insist(fd >= 0, "open(%s) failed: %s", config->path, strerror(errno));

  char *buf;
  ssize_t bytes;
  buf = calloc(BUFFERSIZE, sizeof(char));

  struct backoff sleeper;
  backoff_init(&sleeper, &min_sleep, &max_sleep);

  void *socket = zmq_socket(config->zmq, ZMQ_PUSH);
  insist(socket != NULL, "zmq_socket() failed: %s", strerror(errno));

  int64_t hwm = 100;
  zmq_setsockopt(socket, ZMQ_HWM, &hwm, sizeof(hwm));

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
    bytes = read(fd, buf + offset, BUFFERSIZE - offset - 1);
    if (bytes < 0) {
      /* error, maybe indicate a failure of some kind. */
      break;
    } else if (bytes == 0) {
      backoff(&sleeper);
    } else {
      backoff_clear(&sleeper);

      /* For each line, emit. Save the remainder */
      char *line;
      char *septok = buf;
      char *start = NULL;
      json_t *line_obj = json_string(NULL);
      json_object_set(event, "line", line_obj);
      while (start = septok, (line = strsep(&septok, "\n")) != NULL) {
        if (septok == NULL) {
          /* last token found, no terminator though */
          offset = start - line;
          memmove(buf + offset, buf, strlen(buf + offset));
        } else {
          /* emit line as an event */
          char *serialized;
          size_t serialized_len;
          /* TODO(sissel): skip if line_obj is null (means line was invalid UTF-8) */
          /* TODO(sissel): include file offset */

#ifdef SERIALIZE_JSON
          /* serialize to json */
          json_string_set(line_obj, line);
          serialized = json_dumps(event, 0);
          serialized_len = strlen(serialized_len);
          //json_object_del(line_obj, "line");
          //json_decref(line_obj);
#endif

          /** SERIALIZING MY WAY */
#ifdef SERIALIZE
          int32_t length = 0;
          char *pos; /* moving pointer for writing */
          serialized_len = sizeof(length) + hostname_len + sizeof(length) + path_len 
            + sizeof(length) + (septok-start);
          serialized = malloc(serialized_len);
          pos = serialized;

          /* write length + hostname */
          length = ntohl(hostname_len);
          memcpy(pos, &length, sizeof(length)); pos += sizeof(length);
          memcpy(pos, hostname, hostname_len); pos += hostname_len;

          /* write length + file path */
          length = ntohl(path_len);
          memcpy(pos, &length, sizeof(length)); pos += sizeof(length);
          memcpy(pos, config->path, path_len); pos += path_len;

          /* write length + line */
          length = ntohl(septok - start);
          memcpy(pos, &length, sizeof(length)); pos += sizeof(length);
          memcpy(pos, line, septok - start); pos += septok - start;
#endif

          serialized = line;
          serialized_len = septok - start;

          zmq_msg_t event;
          //zmq_msg_init_data(&event, serialized, serialized_len, free2, NULL);
          zmq_msg_init_data(&event, serialized, serialized_len, NULL, NULL);
          rc = zmq_send(socket, &event, 0);
          insist(rc == 0, "zmq_send(event) failed: %s", zmq_strerror(rc));

          zmq_msg_close(&event);
        }
      } /* for each token */

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


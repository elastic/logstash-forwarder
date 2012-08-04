#define _BSD_SOURCE
#include <string.h> /* for strsep, strerror, etc */
#include <errno.h> /* for errno */
#include <fcntl.h> /* for open(2) */
#include <unistd.h> /* for close, etc */
#include <arpa/inet.h> /* for ntohl */
#include <stdio.h> /* printf and friends */
#include <zmq.h> /* zeromq messaging library */
#include "str.h" /* dynamic string library */
#include "proto.h" /* lumberjack wire format serialization */

#include "harvester.h"
#include "backoff.h"
#include "insist.h"

#ifdef __MACH__
/* OS X is dumb, or I am dumb, or we are both dumb. I don't know anymore,
 * but I need to declare these explicitly even though they are defined
 * in string.h, unistd.h respectively */
extern char *strsep(char **stringp, const char *delim);
extern int gethostname(char *name, size_t namelen);
#endif

#define EMITTER_SOCKET "inproc://emitter"
#define BUFFERSIZE 16384

static struct timespec min_sleep = { 0, 10000000 }; /* 10ms */
static struct timespec max_sleep = { 15, 0 }; /* 15 */

/* A free function that simply calls free(3) for zmq_msg */
//static inline void free2(void *data, void __attribute__((__unused__)) *hint) {
  //free(data);
//} /* free2 */

/* A free function for zmq_msg's with 'struct str' objects */
static inline void my_str_free(void __attribute__((__unused__)) *data, void *hint) {
  str_free((struct str *)hint);
} /* my_str_free */

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
  insist(fd >= 0, "open(%s) failed: %s", config->path, strerror(errno));

  path_len = strlen(config->path);

  struct kv event[] = {
    { "file", 4, config->path, path_len },
    { "host", 4, hostname, hostname_len },
    { "line", 4, NULL, 0 }, /* will fill this in later */
  };

  char *buf;
  ssize_t bytes;
  buf = calloc(BUFFERSIZE, sizeof(char));

  struct backoff sleeper;
  backoff_init(&sleeper, &min_sleep, &max_sleep);

  void *socket = zmq_socket(config->zmq, ZMQ_PUSH);
  insist(socket != NULL, "zmq_socket() failed: %s", strerror(errno));

  int64_t hwm = 100;
  zmq_setsockopt(socket, ZMQ_HWM, &hwm, sizeof(hwm));

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
      while (start = septok, (line = strsep(&septok, "\n")) != NULL) {
        if (septok == NULL) {
          /* last token found, no terminator though */
          offset = start - line;
          memmove(buf + offset, buf, strlen(buf + offset));
        } else {
          /* emit line as an event */
          size_t line_len = septok - start;
          struct str *serialized;

          /* Set the line */
          event[2].value = line;
          event[2].value_len = line_len;

          /* pack using lumberjack data payload */
          serialized = lumberjack_kv_pack(event, 3 /* 3 elements */);

          zmq_msg_t event;
          zmq_msg_init_data(&event, str_data(serialized), str_length(serialized), my_str_free, serialized);
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


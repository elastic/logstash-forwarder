#define _BSD_SOURCE
#include <string.h> /* for strsep, strerror, etc */
#include <errno.h> /* for errno */
#include <fcntl.h> /* for open(2) */
#include <unistd.h> /* for close, etc */
#include <arpa/inet.h> /* for ntohl */
#include <stdio.h> /* printf and friends */
#include "zmq.h" /* zeromq messaging library */
#include "str.h" /* dynamic string library */
#include "proto.h" /* lumberjack wire format serialization */
#include <sys/stat.h>
#include "jemalloc/jemalloc.h"

#include "harvester.h"
#include "backoff.h"
#include "insist.h"
#include "sleepdefs.h"
#include "flog.h"
#include "zmq_compat.h"

#ifdef __MACH__
/* OS X is dumb, or I am dumb, or we are both dumb. I don't know anymore,
 * but I need to declare these explicitly even though they are defined
 * in string.h, unistd.h respectively */
extern char *strsep(char **stringp, const char *delim);
extern int gethostname(char *name, size_t namelen);
#endif

#define EMITTER_SOCKET "inproc://emitter"
#define BUFFERSIZE 16384

/* A free function that simply calls free(3) for zmq_msg */
//static inline void free2(void *data, void __attribute__((__unused__)) *hint) {
  //free(data);
//} /* free2 */

/* A free function for zmq_msg's with 'struct str' objects */
static inline void my_str_free(void __attribute__((__unused__)) *data, void *hint) {
  str_free((struct str *)hint);
} /* my_str_free */

static void track_rotation(int *fd, const char *path);

void *harvest(void *arg) {
  struct harvest_config *config = arg;
  int fd;
  int rc;
  char hostname[200];
  size_t hostname_len, path_len;
  
  /* Make this so we only call it once. */
  gethostname(hostname, sizeof(hostname));
  hostname_len = strlen(hostname);

  if (strcmp(config->path, "-") == 0) {
    /* path is '-', use stdin */
    fd = 0;
  } else {
    fd = open(config->path, O_RDONLY);
    insist(fd >= 0, "open(%s) failed: %s", config->path, strerror(errno));
    /* Start at the end of the file */
    off_t seek_ret = lseek(fd, 0, SEEK_END);
    insist(seek_ret >= 0, "lseek(%s, 0, SEEK_END) failed: %s",
           config->path, strerror(errno));
  }
  path_len = strlen(config->path);

  struct kv *event = calloc(3 + config->fields_len, sizeof(struct kv));

  /* will fill the 'line' value in later for each line read */
  event[0].key = "line"; event[0].key_len = 4;
  event[0].value = NULL; event[0].value_len = 0;
  event[1].key = "file"; event[1].key_len = 4;
  event[1].value = config->path; event[1].value_len = path_len;
  event[2].key = "host"; event[2].key_len = 4;
  event[2].value = hostname; event[2].value_len = hostname_len;
  for (size_t i = 0; i < config->fields_len; i++) {
    memcpy(&event[i + 3], &config->fields[i], sizeof(struct kv));
  }

  char *buf;
  ssize_t bytes;
  buf = calloc(BUFFERSIZE, sizeof(char));

  void *socket = zmq_socket(config->zmq, ZMQ_PUSH);
  insist(socket != NULL, "zmq_socket() failed: %s", strerror(errno));

  int64_t hwm = 100;
  //zmq_setsockopt(socket, ZMQ_HWM, &hwm, sizeof(hwm));
  zmq_compat_set_sendhwm(socket, hwm);

  /* Wait for the zmq endpoint to be up (wait for connect to succeed) */
  struct backoff sleeper;
  backoff_init(&sleeper, &MIN_SLEEP, &MAX_SLEEP);
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

  int offset = 0;
  for (;;) {
    flog_if_slow(stdout, 0.250, {
      bytes = read(fd, buf + offset, BUFFERSIZE - offset - 1);
    }, "read of %d bytes (got %d bytes) on '%s'",
    BUFFERSIZE - offset - 1, bytes, config->path);

    offset += bytes;
    if (bytes < 0) {
      /* error, maybe indicate a failure of some kind. */
      printf("read(%d '%s') failed: %s\n", fd,
             config->path, strerror(errno));
      break;
    } else if (bytes == 0) {
      backoff(&sleeper);
      if (strcmp(config->path, "-") == 0) {
        /* stdin gave EOF, close out. */
        break;
      }
      track_rotation(&fd, config->path);
    } else {
      /* Data read, handle it! */
      backoff_clear(&sleeper);
      /* For each line, emit. Save the remainder */
      char *line;
      char *septok = buf;
      char *start = NULL;
      while (start = septok, (line = strsep(&septok, "\n")) != NULL) {
        if (septok == NULL) {
          /* last token found, no terminator though */
          offset = offset - (line - buf);
          memmove(buf, line, strlen(line));
        } else {
          /* emit line as an event */
          /* 'septok' points at the start of the next token, so subtract one. */
          size_t line_len = septok - start - 1;
          struct str *serialized;

          /* Set the line */
          event[0].value = line;
          event[0].value_len = line_len;

          /* pack using lumberjack data payload */
          serialized = lumberjack_kv_pack(event, 3 + config->fields_len);

          zmq_msg_t event;
          zmq_msg_init_data(&event, str_data(serialized), str_length(serialized),
                            my_str_free, serialized);
          flog_if_slow(stdout, 0.250, {
            //rc = zmq_send(socket, &event, 0);
            rc = zmq_compat_sendmsg(socket, &event, 0);
          }, "zmq_send (harvesting file '%s')", config->path);
          insist(rc == 0, "zmq_send(event) failed: %s", zmq_strerror(rc));
          zmq_msg_close(&event);
        }
      } /* for each token */
    }
  } /* loop forever, reading from a file */

  free(arg); /* allocated by the main method, up to us to free */
  close(fd);

  return NULL;
} /* harvest */

void track_rotation(int *fd, const char *path) {
  struct stat pathstat, fdstat;
  int rc;
  fstat(*fd, &fdstat);
  rc = stat(path, &pathstat);
  if (rc == -1) {
    /* error stat'ing the file path, restart loop and try again */
    return;
  }

  if (pathstat.st_dev != fdstat.st_dev || pathstat.st_ino != fdstat.st_ino) {
    /* device or inode number changed, this file was renamed or rotated. */
    rc = open(path, O_RDONLY);
    if (rc == -1) {
      /* Error opening file, restart loop and try again. */
      return;
    }
    close(*fd);
    /* start reading the new file! */
    *fd = rc; 
  } else if (fdstat.st_size < lseek(*fd, 0, SEEK_CUR)) {
    /* the file was truncated, jump back to the beginning */
    lseek(*fd, 0, SEEK_SET);
  }
} /* track_rotation */

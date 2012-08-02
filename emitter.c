#define _BSD_SOURCE
#include "emitter.h"
#include <time.h>
#include <zmq.h>
#include "insist.h"
#include <errno.h>
#include <string.h>

void *emitter(void *arg) {
  struct emitter_config *config = arg;
  int rc;

  void *socket = zmq_socket(config->zmq, ZMQ_PULL);
  insist(socket != NULL, "zmq_socket() failed: %s", strerror(errno));
  int64_t hwm = 100;
  zmq_setsockopt(socket, ZMQ_HWM, &hwm, sizeof(hwm));
  rc = zmq_bind(socket, config->zmq_endpoint);
  insist(rc != -1, "zmq_bind(%s) failed: %s", config->zmq_endpoint,
         zmq_strerror(errno));

  struct timespec start;
  clock_gettime(CLOCK_MONOTONIC, &start);
  for (int count = 0; ;count++) {
    zmq_msg_t message;
    rc = zmq_msg_init(&message);
    insist(rc == 0, "zmq_msg_init failed");
    if (count == 1000000) {
      struct timespec now;
      clock_gettime(CLOCK_MONOTONIC, &now);
      double s = (start.tv_sec + 0.0) + (start.tv_nsec / 1000000000.0);
      double n = (now.tv_sec + 0.0) + (now.tv_nsec / 1000000000.0);
      printf("Rate: %f\n", (count + 0.0) / (n - s));
      clock_gettime(CLOCK_MONOTONIC, &start);
      count = 0;
    }
    rc = zmq_recv(socket, &message, 0);
    insist(rc == 0, "zmq_recv(%s) failed (returned %d): %s",
           config->zmq_endpoint, rc, zmq_strerror(errno));
    //printf("received: %.*s\n", (int)zmq_msg_size(&message),
           //(char *)zmq_msg_data(&message));

    /* TODO(sissel): ship this out to a remote server */
    zmq_msg_close(&message);
  }
} /* emitter */

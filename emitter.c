#define _BSD_SOURCE
#include "emitter.h"
#include <zmq.h>
#include "insist.h"
#include <errno.h>
#include <string.h>

void *emitter(void *arg) {
  struct emitter_config *config = arg;
  int rc;

  void *socket = zmq_socket(config->zmq, ZMQ_PULL);
  insist(socket != NULL, "zmq_socket() failed: %s", strerror(errno));
  int64_t hwm = 1;
  zmq_setsockopt(socket, ZMQ_HWM, &hwm, sizeof(hwm));
  rc = zmq_bind(socket, config->zmq_endpoint);
  insist(rc != -1, "zmq_bind(%s) failed: %s", config->zmq_endpoint,
         zmq_strerror(errno));

  for (;;) {
    zmq_msg_t message;
    rc = zmq_msg_init(&message);
    insist(rc == 0, "zmq_msg_init failed");
    rc = zmq_recv(socket, &message, 0);
    insist(rc == 0, "zmq_recv(%s) failed (returned %d): %s",
           config->zmq_endpoint, rc, zmq_strerror(errno));
    //printf("received: %.*s\n", (int)zmq_msg_size(&message),
           //(char *)zmq_msg_data(&message));

    /* TODO(sissel): ship this out to a remote server */
    zmq_msg_close(&message);
  }
} /* emitter */

#include <errno.h>
#include <stdint.h> /* C99 for int64_t */
#include <string.h>
#include <unistd.h>
#include <zmq.h>
#include "ring.h"
#include "emitter.h"
#include "insist.h"
#include "proto.h"
#include "backoff.h"
#include "clock_gettime.h"

static struct timespec MIN_SLEEP = { 0, 10000000 }; /* 10ms */
static struct timespec MAX_SLEEP = { 15, 0 }; /* 15 */

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
  //long count = 0;

  struct backoff sleeper;
  backoff_init(&sleeper, &MIN_SLEEP, &MAX_SLEEP);

  struct lumberjack *lumberjack;
  lumberjack = lumberjack_new(config->host, config->port);
  insist(lumberjack != NULL, "lumberjack_new failed");

  if (config->ssl_ca_path != NULL) {
    rc = lumberjack_set_ssl_ca(lumberjack, config->ssl_ca_path);
    if (rc != 0) {
      printf("lumberjack_set_ssl_ca failed. Is '%s' valid?\n", config->ssl_ca_path);
      return NULL;
    }
  }

  long count = 0;
  for (;;) {
    /* Receive an event from a harvester and put it in the queue */
    zmq_msg_t message;

    rc = zmq_msg_init(&message);
    insist(rc == 0, "zmq_msg_init failed");
    //printf("waiting for zmq\n");
    rc = zmq_recv(socket, &message, 0);
    insist(rc == 0, "zmq_recv(%s) failed (returned %d): %s",
           config->zmq_endpoint, rc, zmq_strerror(errno));

    /* Write the data over lumberjack. This will handle any
     * connection/reconnection/ack issues */
    lumberjack_send_data(lumberjack, zmq_msg_data(&message),
                         zmq_msg_size(&message));
    zmq_msg_close(&message);

    /* Stats for debugging */
    count++;
    if (count == 10000) {
      struct timespec now;
      clock_gettime(CLOCK_MONOTONIC, &now);
      double s = (start.tv_sec + 0.0) + (start.tv_nsec / 1000000000.0);
      double n = (now.tv_sec + 0.0) + (now.tv_nsec / 1000000000.0);
      fprintf(stderr, "Rate: %f\n", (count + 0.0) / (n - s));
      clock_gettime(CLOCK_MONOTONIC, &start);
      count = 0;
    }
  } /* forever */
} /* emitter */


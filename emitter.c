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

  /* Seed the RNG so we can pick a random starting sequence number */
  srand(time(NULL));
  uint32_t sequence = 0; //rand();

  struct timespec start;
  clock_gettime(CLOCK_MONOTONIC, &start);
  //long count = 0;

  struct ring *ring = ring_new_size(2048); /* power of 2 is required*/

  struct backoff sleeper;
  backoff_init(&sleeper, &MIN_SLEEP, &MAX_SLEEP);

  struct lumberjack *lumberjack;
  lumberjack = lumberjack_new("localhost", 1234);
  insist(lumberjack != NULL, "lumberjack_new failed");

  long count = 0;
  for (;;) {
    if (!lumberjack_connected(lumberjack)) {
      backoff(&sleeper);

      rc = lumberjack_connect(lumberjack);
      if (rc != 0) {
        printf("Connection attempt to %s:%hd failed: %s\n",
               lumberjack->host, lumberjack->port, strerror(errno));
        continue;
      }
      backoff_clear(&sleeper);

      /* New connection, anything in the ring buffer is assumed to be
       * un-acknowledged. Send it. */
      for (int i = 0, count = ring_count(ring); i < count; i++) {
        struct str *frame;
        rc = ring_peek(ring, i, (void **)&frame);
        insist(rc == RING_OK, "ring_peek(%d) failed unexpectedly: %d\n", i, rc);
        rc = lumberjack_write(lumberjack, frame);
        if (rc != 0) {
          /* write failed, break and reconnect */
          break;
        }
      }
    }

    if (ring_is_full(ring)) {
      /* Too many data frames waiting on acknowledgement, read acks until it
       * would block ? */
      uint32_t ack;

      rc = lumberjack_read_ack(lumberjack, &ack);
      if (rc < 0) {
        /* error */
        printf("lumberjack_read_ack failed: %s\n", strerror(errno));
        lumberjack_disconnect(lumberjack);
        backoff(&sleeper);
        continue;
      }

      //printf("Got ack for %d\n", ack);

      /* TODO(sissel): Verify this is even a sane ack */
      struct str *frame;
      uint32_t cur_seq;
      /* Clear anything in the ring with a sequence less than the one just acked */
      for (int i = 0, count = ring_count(ring); i < count; i++) {
        ring_peek(ring, 0, (void **)&frame);
        /* this is a silly way, but since the ring only stores strings right now */
        memcpy(&cur_seq, str_data(frame) + 2, sizeof(uint32_t));
        cur_seq = ntohl(cur_seq);

        if (cur_seq <= ack) {
          //printf("bulk ack: %d\n", cur_seq);
          ring_pop(ring, NULL); /* don't care to retrieve it */
          str_free(frame);
        } else {
          break;
        }
      }
    } else {
      /* Receive an event from a harvester and put it in the queue */
      zmq_msg_t message;
      struct str *frame;

      rc = zmq_msg_init(&message);
      insist(rc == 0, "zmq_msg_init failed");
      //printf("waiting for zmq\n");
      rc = zmq_recv(socket, &message, 0);
      insist(rc == 0, "zmq_recv(%s) failed (returned %d): %s",
             config->zmq_endpoint, rc, zmq_strerror(errno));

      /* Build a lumberjack 'data' frame payload, put it in the ring buffer */
      sequence++;
      frame = lumberjack_encode_data(sequence, zmq_msg_data(&message),
                                     zmq_msg_size(&message));
      rc = ring_push(ring, frame);
      insist(rc == RING_OK, "ring_push failed (returned %d, expected RING_OK(%d)",
             rc, RING_OK);

      //printf("seq: %d\n", sequence);
      /* Write a lumberjack frame, this will block until the full write
       * completes or errors. On error, it will disconnect. */

      /* TODO(sissel): SIGPIPE here sometimes. need to avoid it. */
      rc = lumberjack_write(lumberjack, frame);

      zmq_msg_close(&message);


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
    }
  } /* forever */
} /* emitter */


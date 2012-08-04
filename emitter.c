#include "emitter.h"
#include <zmq.h>
#include "insist.h"
#include <errno.h>
#include <stdint.h> /* C99 for int64_t */
#include <string.h>
#include <time.h> /* struct timespec, clock_gettime */
#include <unistd.h>

// copied mostly from https://gist.github.com/1087739
/* OS X doesn't have clock_gettime, sigh. */
#ifdef __MACH__
#include <mach/clock.h>
#include <mach/mach.h>

typedef int clockid_t;
#define CLOCK_MONOTONIC 1
long clock_gettime(clockid_t __attribute__((unused)) which_clock, struct timespec *tp) {
  clock_serv_t cclock;
  mach_timespec_t mts;
  host_get_clock_service(mach_host_self(), REALTIME_CLOCK, &cclock);
  clock_get_time(cclock, &mts);
  mach_port_deallocate(mach_task_self(), cclock);
  tp->tv_sec = mts.tv_sec;
  tp->tv_nsec = mts.tv_nsec;
  return 0; /* success, according to clock_gettime(3) */
}
#endif
// end gist copy

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

  //srand(time(NULL));

  struct timespec start;
  clock_gettime(CLOCK_MONOTONIC, &start);
  long count = 0;

  for (;;) {
    /* TODO(sissel): If buffer is full and this is not a fresh connection,
     * block for acks */
    /* TODO(sissel): If buffer is not empty, write one event. */
      /* TODO(sissel): write frame header (version + frame type 'D') */
      /* TODO(sissel): write sequence number */
      /* TODO(sissel): write event payload */
    /* TODO(sissel): On any write/connect error, block until reconnected. 
     * When reconnected, restart this loop to flush buffer. */

    /* Receive an event from a harvester and put it in the queue */
    zmq_msg_t message;
    rc = zmq_msg_init(&message);
    insist(rc == 0, "zmq_msg_init failed");
    rc = zmq_recv(socket, &message, 0);
    insist(rc == 0, "zmq_recv(%s) failed (returned %d): %s",
           config->zmq_endpoint, rc, zmq_strerror(errno));

    //write(1, zmq_msg_data(&message), zmq_msg_size(&message));
    //write(1, "\n", 1);


    /* TODO(sissel): pick sequence number */
    /* TODO(sissel): put the event into the ring buffer */

    /* TODO(sissel): ship this out to a remote server */
    zmq_msg_close(&message);

    count++;
    if (count == 1000000) {
      struct timespec now;
      clock_gettime(CLOCK_MONOTONIC, &now);
      double s = (start.tv_sec + 0.0) + (start.tv_nsec / 1000000000.0);
      double n = (now.tv_sec + 0.0) + (now.tv_nsec / 1000000000.0);
      fprintf(stderr, "Rate: %f\n", (count + 0.0) / (n - s));
      clock_gettime(CLOCK_MONOTONIC, &start);
      count = 0;
    }
  }
} /* emitter */

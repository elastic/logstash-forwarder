#include "emitter.h"
#include <zmq.h>
#include "insist.h"
#include <errno.h>
#include <stdint.h> /* C99 for int64_t */
#include <string.h>

#ifdef __MACH__
// copied mostly from https://gist.github.com/1087739
/* OS X doesn't have clock_gettime, sigh */
#include <mach/clock.h>
#include <mach/mach.h>

typedef int clockid_t;
#define CLOCK_MONOTONIC 1
long clock_gettime(clockid_t __attribute__((unused)) which_clock, struct timespec *tp) {
  clock_serv_t cclock;
  mach_timespec_t mts;
  host_get_clock_service(mach_host_self(), CALENDAR_CLOCK, &cclock);
  clock_get_time(cclock, &mts);
  mach_port_deallocate(mach_task_self(), cclock);
  tp->tv_sec = mts.tv_sec;
  tp->tv_nsec = mts.tv_nsec;
  return 0; /* success, according to clock_gettime(3) */
}
#else
#include <time.h>
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

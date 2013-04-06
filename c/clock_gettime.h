#ifndef _CLOCK_GETTIME_H_
#define _CLOCK_GETTIME_H_
#include <time.h> /* struct timespec, clock_gettime */

// copied mostly from https://gist.github.com/1087739
/* OS X doesn't have clock_gettime, sigh. */
#ifdef __MACH__
#include <mach/clock.h>
#include <mach/mach.h>

typedef int clockid_t;
#define CLOCK_MONOTONIC 1
static long clock_gettime(clockid_t __attribute__((unused)) which_clock, struct timespec *tp) {
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

#endif /* _CLOCK_GETTIME_H_ */

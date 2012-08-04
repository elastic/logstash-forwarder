//#define _POSIX_C_SOURCE 199309L /* for struct timespec */
#include <stdio.h>
#include "backoff.h"

#define MAX_TV_NSEC 999999999L

static inline void timespec_double(struct timespec *t) {
  /* Exponential backoff */
  t->tv_sec <<= 1;
  t->tv_nsec <<= 1;
  if (t->tv_nsec > MAX_TV_NSEC) {
    /* handle carry/overflow of tv_nsec */
    t->tv_nsec -= (MAX_TV_NSEC + 1);
    t->tv_sec += 1;
  }
} /* timespec_double */

static inline long timespec_compare(struct timespec *a, struct timespec *b) {
  time_t val;
  val = a->tv_sec - b->tv_sec;
  if (val != 0) {
    return val;
  } 
  return a->tv_nsec - b->tv_nsec;
} /* timespec_compare */

static inline void timespec_copy(struct timespec *source, struct timespec *dest) {
  /* TODO(sissel): Could use memcpy here instead... */
  dest->tv_sec = source->tv_sec;
  dest->tv_nsec = source->tv_nsec;
} /* timespec_copy */

inline void backoff_clear(struct backoff *b) {
  timespec_copy(&b->min, &b->sleep);
} /* backoff_clear */

inline void backoff_init(struct backoff *b, struct timespec *min,
                         struct timespec *max) {
  timespec_copy(min, &b->min);
  timespec_copy(max, &b->max);
  backoff_clear(b);
} /* backoff_init */

inline void backoff(struct backoff *b) {
  //printf("Sleeping %ld.%09ld\n", b->sleep.tv_sec, b->sleep.tv_nsec);
  nanosleep(&b->sleep, NULL);

  /* Exponential backoff */
  timespec_double(&b->sleep);
  //printf("Candidate vs max: %ld.%09ld vs %ld.%09ld: %ld\n",
         //b->sleep.tv_sec, b->sleep.tv_nsec,
         //b->max.tv_sec, b->max.tv_nsec,
         //timespec_compare(&b->sleep, &b->max));
  //printf("tv_sec: %ld\n", b->sleep.tv_sec - b->max.tv_sec);
  //printf("tv_nsec: %ld\n", b->sleep.tv_nsec - b->max.tv_nsec);

  /* Cap at 'max' if sleep time exceeds it */
  if (timespec_compare(&b->sleep, &b->max) > 0) {
    timespec_copy(&b->max, &b->sleep);
  }
} /* backoff_sleep */

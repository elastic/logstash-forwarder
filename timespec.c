#include "timespec.h"
const struct timespec TIME_ZERO = { 0, 0 };

inline void timespec_double(struct timespec *t) {
  /* Exponential backoff */
  t->tv_sec <<= 1;
  t->tv_nsec <<= 1;
  if (t->tv_nsec > MAX_TV_NSEC) {
    /* handle carry/overflow of tv_nsec */
    t->tv_nsec -= (MAX_TV_NSEC + 1);
    t->tv_sec += 1;
  }
} /* timespec_double */

inline long timespec_compare(struct timespec *a, struct timespec *b) {
  time_t val;
  val = a->tv_sec - b->tv_sec;
  if (val != 0) {
    return val;
  } 
  return a->tv_nsec - b->tv_nsec;
} /* timespec_compare */

inline void timespec_copy(struct timespec *source, struct timespec *dest) {
  /* TODO(sissel): Could use memcpy here instead... */
  dest->tv_sec = source->tv_sec;
  dest->tv_nsec = source->tv_nsec;
} /* timespec_copy */

inline void timespec_subtract(struct timespec *a, struct timespec *b,
                              struct timespec *result) {
  result->tv_nsec = a->tv_nsec - b->tv_nsec;
  result->tv_sec = a->tv_sec - b->tv_sec;

  if (result->tv_nsec < 0) {
    /* Handle carry */
    result->tv_nsec += 1000000000L;
    result->tv_sec -= 1;
  }
} 

inline void timespec_add(struct timespec *a, struct timespec *b,
                         struct timespec *result) {
  result->tv_nsec = a->tv_nsec + b->tv_nsec;
  result->tv_sec = a->tv_sec + b->tv_sec;

  if (result->tv_nsec > MAX_TV_NSEC) {
    /* Handle carry */
    result->tv_nsec -= 1000000000L;
    result->tv_sec += 1;
  }
} 

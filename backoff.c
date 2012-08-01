#include "backoff.h"
#include <stdio.h>
#include <unistd.h>

inline void backoff_init(struct backoff *b, useconds_t min, useconds_t max) {
  b->max = max;
  b->min = min;
  backoff_clear(b);
} /* backoff_init */

inline void backoff(struct backoff *b) {
  //printf("Sleeping %f seconds\n", b->time / 1000000.0);
  usleep(b->time);

  /* Exponential backoff */
  b->time <<= 1;

  /* Cap at 'max' time sleep */
  if (b->time > b->max) {
    b->time = b->max;
  }
} /* backoff_sleep */

inline void backoff_clear(struct backoff *b) {
  b->time = b->min; /* 1000 microseconds == 1ms */
} /* backoff_clear */

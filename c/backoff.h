#ifndef _BACKOFF_H_
#define _BACKOFF_H_

#include <time.h>

struct backoff {
  struct timespec max;
  struct timespec min;
  struct timespec sleep;
};

/* Initialize a backoff struct with a max value */
void backoff_init(struct backoff *b, struct timespec *min, struct timespec *max);

/* Execute a backoff. This will sleep for a time.
 * The next backoff() call will sleep twice as long (or the max value,
 * whichever is smaller) */
void backoff(struct backoff *b);

/* Reset the next backoff() call to sleep the minimum (1ms) */
void backoff_clear(struct backoff *b);
#endif /* _BACKOFF_H_ */

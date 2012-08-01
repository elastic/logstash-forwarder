#ifndef _BACKOFF_H_
#define _BACKOFF_H_

#include <sys/types.h>

struct backoff {
  useconds_t max;
  useconds_t min;
  useconds_t time;
};

/* Initialize a backoff struct with a max value */
void backoff_init(struct backoff *b, useconds_t min, useconds_t max);

/* Execute a backoff. This will sleep for a time.
 * The next backoff() call will sleep twice as long (or the max value,
 * whichever is smaller) */
void backoff(struct backoff *b);

/* Reset the next backoff() call to sleep the minimum (1ms) */
void backoff_clear(struct backoff *b);
#endif /* _BACKOFF_H_ */

#ifndef _TIMESPEC_H_
#define _TIMESPEC_H_
#include <time.h>
#define MAX_TV_NSEC 999999999L
void timespec_double(struct timespec *t);
long timespec_compare(struct timespec *a, struct timespec *b);
void timespec_copy(struct timespec *source, struct timespec *dest);
void timespec_subtract(struct timespec *a, struct timespec *b,
                       struct timespec *result);
void timespec_add(struct timespec *a, struct timespec *b,
                  struct timespec *result);

#endif /* _TIMESPEC_H_ */

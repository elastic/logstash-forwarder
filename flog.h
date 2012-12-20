#ifndef _FLOG_H_
#define _FLOG_H_
#include <stdio.h> /* for FILE */
#include <sys/time.h> /* for struct timeval */
void flog(FILE *stream, const char *format, ...);
double duration(struct timeval *start);

#define flog_if_slow(stream, max_duration, block, format, args...) \
{ \
  struct timeval __start; \
  gettimeofday(&__start, NULL); \
  { \
    block \
  } \
  double __duration = duration(&__start); \
  if (__duration >= max_duration) { \
    flog(stream, "slow operation (%.3f seconds): " format , __duration, args); \
  } \
}

#endif /* _FLOG_H_ */

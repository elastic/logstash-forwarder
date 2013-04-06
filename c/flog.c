#include <stdio.h> /* for FILE, sprintf, fprintf, etc */
#include <time.h> /* for struct tm, localtime_r */
#include <sys/time.h> /* for gettimeofday */
#include <stdarg.h> /* for va_start, va_end */

void flog(FILE *stream, const char *format, ...) {
  va_list args;
  struct timeval tv;
  struct tm tm;
  char timestamp[] = "YYYY-MM-ddTHH:mm:ss.SSS+0000";
  gettimeofday(&tv, NULL);

  /* convert to time to 'struct tm' for use with strftime */
  localtime_r(&tv.tv_sec, &tm);

  /* format the time */
  strftime(timestamp, sizeof(timestamp), "%Y-%m-%dT%H:%M:%S.000%z", &tm);

  /* add in milliseconds, since strftime() can't do that */
  /* '20' is the string offset of the millisecond value in our timestamp */
  /* we have to include 'timestamp + 23' to keep the timezone value */
  sprintf(timestamp + 20, "%03ld%s", tv.tv_usec / 1000, timestamp + 23);

  /* print the timestamp */
  fprintf(stream, "%.28s ", timestamp); /* 28 is the length of the timestamp */

  /* print the log message */
  va_start(args, format);
  vfprintf(stream, format, args);
  va_end(args);

  /* print a newline */
  fprintf(stream, "\n");
} /* flog */

double duration(struct timeval *start) {
  struct timeval tv;
  gettimeofday(&tv, NULL); /* what time is it now? */

  tv.tv_sec -= start->tv_sec;
  tv.tv_usec -= start->tv_usec;

  if (tv.tv_usec < 0) {
    tv.tv_sec -= 1;
    tv.tv_usec += 1000000L;
  }
  return tv.tv_sec + ((double)tv.tv_usec / 1000000.0);
} /* duration */

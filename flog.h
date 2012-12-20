#ifndef _FLOG_H_
#define _FLOG_H_
#include <stdio.h> /* for FILE */
#include <sys/time.h> /* for struct timeval */
void flog(FILE *stream, const char *format, ...);
double duration(struct timeval *start);

#endif /* _FLOG_H_ */

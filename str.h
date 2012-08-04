#ifndef _STR_H_
#define _STR_H_
#include <stdint.h>
#include <sys/types.h>

struct str {
  char *data;
  size_t data_len;
  size_t data_size;
}; /* struct str */

struct str *str_new(void);
void str_free(struct str *str);
void str_grow(struct str *str);
size_t str_length(struct str *str);
char *str_value(struct str *str);
void str_append(struct str *str, const char *data, size_t length);
void str_append_uint32(struct str *str, uint32_t value);

#endif /* _STR_H_ */

#ifndef _STR_H_
#define _STR_H_
#include <stdint.h>
#include <sys/types.h>

struct str {
  size_t data_len;
  size_t data_size;
  char *data;
}; /* struct str */

/* Make a new str. */
struct str *str_new(void);
struct str *str_new_size(size_t size);

/* Free a str */
void str_free(struct str *str);

/* grow a string; doubles the storage size */
void str_grow(struct str *str);

size_t str_length(struct str *str);
size_t str_size(struct str *str);
char *str_data(struct str *str);
void str_append(struct str *str, const char *data, size_t length);
void str_append_str(struct str *dst_str, struct str *src_str);
void str_append_uint32(struct str *str, uint32_t value);
void str_append_char(struct str *str, char value);
void str_truncate(struct str *str);

#endif /* _STR_H_ */

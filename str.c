#include <arpa/inet.h> /* for htonl */
#include <string.h> /* for memmove, etc */
#include <stdlib.h> /* for calloc, realloc, etc */
#include "str.h"

inline struct str *str_new(void) {
  return str_new_size(20); /* default small size */
} /* str */

inline struct str *str_new_size(size_t size) {
  struct str *str;
  str = malloc(sizeof(struct str));
  str->data_size = size;
  str->data_len = 0;
  /* benchmark difference in allocating 'str' and its data in the same malloc call */
  //str->data = malloc(str->data_size * sizeof(char));
  str->data = (char *)(str + sizeof(struct str));
  return str;
} /* str */

inline void str_free(struct str *str) {
  //free(str->data);
  free(str);
} /* str */

inline void str_grow(struct str *str) {
  str->data_size <<= 2;
  str->data = realloc(str->data, str->data_size);
} /* str */

inline size_t str_length(struct str *str) {
  return str->data_len;
} /* str_length */

inline char *str_value(struct str *str) {
  return str->data;
} /* str_value */

inline void str_append(struct str *str, const char *data, size_t length) {
  while (str->data_size < (str->data_len + length)) {
    str_grow(str);
  }

  memmove(str->data + str->data_len, data, length);
  str->data_len += length;
} /* str_append */

/* Append an unsigned 32bit unsigned integer to the str written 
 * in network byte order */
inline void str_append_uint32(struct str *str, uint32_t value) {
  value = htonl(value); /* use network byte ordering */
  str_append(str, (char *)&value, sizeof(value));
} /* str_append_uint32 */


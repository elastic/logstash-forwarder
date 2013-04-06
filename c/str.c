#include <arpa/inet.h> /* for htonl */
#include <string.h> /* for memmove, etc */
#include <stdlib.h> /* for calloc, realloc, etc */
#include "str.h"
#include <jemalloc/jemalloc.h>

inline struct str *str_new_size(size_t size) {
  struct str *str;
  str = malloc(sizeof(struct str));
  str->data_size = size;
  str->data_len = 0;
  str->data = malloc(str->data_size * sizeof(char));
  /* We could save ourselves a malloc call by storing the str struct and its
   * data in the same allocation. Needs benchmarking. */
  // Example: str->data = (char *)(str + sizeof(struct str));
  return str;
} /* str */

inline struct str *str_new(void) {
  return str_new_size(20); /* default small size */
} /* str */

inline void str_free(struct str *str) {
  free(str->data);
  free(str);
} /* str */

inline void str_grow(struct str *str) {
  str->data_size <<= 1; /* double the data size */
  str->data = realloc(str->data, str->data_size);
} /* str */

inline size_t str_length(struct str *str) {
  return str->data_len;
} /* str_length */

inline size_t str_size(struct str *str) {
  return str->data_size;
} /* str_size */

inline char *str_data(struct str *str) {
  return str->data;
} /* str_data */

inline void str_append(struct str *str, const char *data, size_t length) {
  /* Grow the string if the new length will be longer than the current
   * allocation */
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

inline void str_append_char(struct str *str, char value) {
  str_append(str, &value, sizeof(value));
} /* str_append_char */

inline void str_truncate(struct str *str) {
  str->data_len = 0;
} /* str_zero */

inline void str_append_str(struct str *dst, struct str *src) {
  str_append(dst, str_data(src), str_length(src));
} /* str_append_str */

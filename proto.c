#include <stdint.h>
#include <sys/types.h>
#include <string.h>
#include <stdlib.h>
#include <unistd.h>
#include <stdio.h>
#include "proto.h"
#include "str.h"

struct kv {
  char *key;
  size_t key_len;
  char *value;
  size_t value_len;
}; /* struct kv */

struct str* lumberjack_kv_pack(struct kv *kv_list, size_t kv_count) {
  struct str *payload;

  /* I experimented with different values here.
   *
   * As this as input:
   *     char log[] = "Aug  3 17:01:05 sandwich ReportCrash[38216]: Removing excessive log: file://localhost/Users/jsissel/Library/Logs/DiagnosticReports/a.out_2012-08-01-164517_sandwich.crash";
   *     char file[] = "/var/log/system.log";
   *     char hostname[] = "sandwich";
   *     struct kv map[] = {
   *       { "line", 4, log, strlen(log) },
   *       { "file", 4, file, strlen(file) },
   *       { "host", 4, hostname, strlen(hostname) }
   *     };
   *
   * Looping doing this:
   *      p = lumberjack_kv_pack(map, 3);
   *      str_free(p);
   *
   * Relative time spent (on 10,000,000 iterations):
   *   - 20 bytes - 3.87
   *   - 100 bytes - 2.99
   *   - 768 bytes - 1.65
   *   - 1008 bytes - 1.65
   *   - 1009 bytes - 1.24
   *   - 1010 bytes - 1.24
   *   - 1024 bytes - 1.24
   *
   */
  payload = str_new_size(1009);

  for (int i = 0; i < kv_count; i++) {
    str_append_uint32(payload, kv_list[i].key_len);
    str_append(payload, kv_list[i].key, kv_list[i].key_len);
    str_append_uint32(payload, kv_list[i].value_len);
    str_append(payload, kv_list[i].value, kv_list[i].value_len);
  }

  return payload;
}

int main(int argc, char *argv[]) {
  struct str *p;
  char log[] = "Aug  3 17:01:05 sandwich ReportCrash[38216]: Removing excessive log: file://localhost/Users/jsissel/Library/Logs/DiagnosticReports/a.out_2012-08-01-164517_sandwich.crash";
  char file[] = "/var/log/system.log";
  char hostname[] = "sandwich";

  struct kv map[] = {
    { "line", 4, log, strlen(log) },
    { "file", 4, file, strlen(file) },
    { "host", 4, hostname, strlen(hostname) }
  };

  int max = atoi(argv[1]);
  for (int i = 0; i < max; i++) {
    p = lumberjack_kv_pack(map, 3);
    str_free(p);
  }
  return 0;
}

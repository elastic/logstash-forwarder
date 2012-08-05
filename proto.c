#include <stdint.h>
#include <sys/types.h>
#include <string.h>
#include <stdlib.h>
#include <unistd.h>
#include <stdio.h>
#include "proto.h"
#include <sys/uio.h> /* for writev */
#include "str.h"
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <errno.h>

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
   *      p = _kv_pack(map, 3);
   *      str_free(p);
   *
   * Relative time spent (on 10,000,000 iterations):
   *   - 768 bytes - 1.65
   *   - 1008 bytes - 1.65
   *   - 1009 bytes - 1.24
   *   - 1010 bytes - 1.24
   *   - 1024 bytes - 1.24
   *
   * Platform tested was OS X 10.7 with XCode's clang/cc
   *   % cc -O4 ...
   *
   * Given that, I pick 1024 (nice round number) for the initial string size
   * for the payload.
   */
  payload = str_new_size(1024);

  str_append_uint32(payload, kv_count);
  for (size_t i = 0; i < kv_count; i++) {
    str_append_uint32(payload, kv_list[i].key_len);
    str_append(payload, kv_list[i].key, kv_list[i].key_len);
    str_append_uint32(payload, kv_list[i].value_len);
    str_append(payload, kv_list[i].value, kv_list[i].value_len);
  }

  return payload;
} /* lumberjack_kv_pack */

/* Example code */
static void example(void) {
  struct str *p;
  char log[] = "Aug  3 17:01:05 sandwich ReportCrash[38216]: Removing excessive log: file://localhost/Users/jsissel/Library/Logs/DiagnosticReports/a.out_2012-08-01-164517_sandwich.crash";
  char file[] = "/var/log/system.log";
  char hostname[] = "sandwich";

  struct kv map[] = {
    { "line", 4, log, strlen(log) },
    { "file", 4, file, strlen(file) },
    { "host", 4, hostname, strlen(hostname) }
  };

  for (int i = 0; i < 1000000; i++) {
    p = lumberjack_kv_pack(map, 3);
    str_free(p);
  }
}

struct lumberjack *lumberjack_new(const char *host, unsigned short port) {
  struct lumberjack *lumberjack;

  lumberjack = malloc(sizeof(struct lumberjack));
  lumberjack->host = host;
  lumberjack->port = port;
  lumberjack->fd = -1;
  return lumberjack;
}

int lumberjack_connect(struct lumberjack *lumberjack) {
  /* TODO(sissel): support ipv6, if anyone ever uses that in production ;) */
  insist(lumberjack != NULL, "lumberjack must not be NULL");
  insist(lumberjack->fd < 0, "already connected (fd %d > 0)", lumberjack->fd);
  insist(lumberjack->host != NULL, "lumberjack host must not be NULL");

  int rc;
  struct hostent *hostinfo = gethostbyname(lumberjack->host);

  if (hostinfo == NULL) {
    /* DNS error, gethostbyname sets h_errno on failure */
    printf("gethostbyname(%s) failed: %s\n", lumberjack->host,  strerror(h_errno));
    return -1;
  }

  /* 'struct hostent' has the list of addresses resolved in 'h_addr_list'
   * It's a null-terminated list, so count how many are there. */
  unsigned int addr_count;
  for (addr_count = 0; hostinfo->h_addr_list[addr_count] != NULL; addr_count++);

  /* hostnames can resolve to multiple addresses, pick one at random. */
  char *address = hostinfo->h_addr_list[rand() % addr_count];

  printf("Connecting to %s(%s):%hd\n",
         lumberjack->host, inet_ntoa(*(struct in_addr *)address),
         lumberjack->port);

  lumberjack->fd = socket(PF_INET, SOCK_STREAM, 0);
  insist(lumberjack->fd >= 0, "socket() failed: %s\n", strerror(errno));

  struct sockaddr_in sockaddr;
  sockaddr.sin_family = PF_INET,
  sockaddr.sin_port = htons(lumberjack->port),
  memcpy(&sockaddr.sin_addr, address, hostinfo->h_length);

  rc = connect(lumberjack->fd, (struct sockaddr *)&sockaddr, sizeof(sockaddr));
  if (rc < 0) {
    lumberjack_disconnect(lumberjack);
    return -1;
  }

  printf("Connected successfully to %s(%s):%hd\n",
         lumberjack->host, inet_ntoa(*(struct in_addr *)address),
         lumberjack->port);

  return 0;
} /* lumberjack_connect */

int lumberjack_write(struct lumberjack *lumberjack, struct str *payload) {
  insist(lumberjack != NULL, "lumberjack must not be NULL");
  insist(payload != NULL, "payload must not be NULL");

  /* writing is an error if you are not connected */
  if (!lumberjack_connected(lumberjack)) {
    return -1;
  }
  size_t remaining = str_length(payload);
  size_t offset = 0;

  ssize_t bytes;
  while (remaining > 0) {
    bytes = write(lumberjack->fd, str_data(payload) + offset, remaining);
    if (bytes < 0) {
      /* error occurred while writing. */
      lumberjack_disconnect(lumberjack);
      return -1;
    }
    remaining -= bytes;
    offset += bytes;
  }

  return 0;
} /* lumberjack_write_v1 */

void lumberjack_disconnect(struct lumberjack *lumberjack) {
  if (lumberjack->fd >= 0) {
    printf("Disconnect requested");
    close(lumberjack->fd);
    lumberjack->fd = -1;
    insist(!lumberjack_connected(lumberjack), 
           "lumberjack_connected() must not return true after a disconnect");
  }
} /* lumberjack_disconnect */
  
struct str *lumberjack_encode_data(uint32_t sequence, const char *payload, size_t payload_len) {
  struct str *data = str_new_size(sizeof(uint32_t) + payload_len);
  str_append_char(data, LUMBERJACK_VERSION_1);
  str_append_char(data, LUMBERJACK_DATA_FRAME);
  str_append_uint32(data, sequence);
  str_append(data, payload, payload_len);
  return data;
} /* lumberjack_data_v1 */

int lumberjack_connected(struct lumberjack *lumberjack) {
  return lumberjack->fd >= 0;
} /* lumberjack_connected */

int lumberjack_read_ack(struct lumberjack *lumberjack, uint32_t *sequence_ret) {
  if (!lumberjack_connected(lumberjack)) {
    printf("NOT CONNECTED\n");
    return -1;
  }

  /* This is a subpar implementation... reading 6 bytes at a time, etc, 
   * but the idea is that you can do bulk acks, so data-to-ack ratio should be
   * high */
  char buf[6]; 
  ssize_t bytes;
  size_t remaining = 6; /* version + frame type + 32bit sequence value */
  size_t offset = 0;
  while (remaining > 0) {
    bytes = read(lumberjack->fd, buf + offset, remaining);
    if (bytes <= 0) {
      /* error(<0) or EOF(0) */
      return -1;
    }
    offset += bytes;
    remaining -= bytes;
  }

  if ((buf[0] != LUMBERJACK_VERSION_1) || (buf[1] != LUMBERJACK_ACK_FRAME)) {
    return -1; /* invalid version or frame type */
  }

  /* bytes 2-6 are the sequence number in network byte-order */
  memcpy(sequence_ret, buf + 2, sizeof(uint32_t));
  *sequence_ret = ntohl(*sequence_ret);

  return 0;
} /* lumberjack_read_ack */

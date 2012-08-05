#ifndef _PROTO_H_
#define _PROTO_H_
#include "str.h"
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <netdb.h>
#include "insist.h"

struct kv {
  char *key;
  size_t key_len;
  char *value;
  size_t value_len;
}; /* struct kv */

struct lumberjack {
  const char *host;
  unsigned short port;

  int fd;
  /* TODO(sissel): Add openssl stuff */
};

#define LUMBERJACK_VERSION_1 '1'
#define LUMBERJACK_DATA_FRAME 'D'
#define LUMBERJACK_ACK_FRAME 'A'
#define LUMBERJACK_WINDOW_SIZE_FRAME 'W'

struct lumberjack *lumberjack_new(const char *host, unsigned short port);

struct str *lumberjack_kv_pack(struct kv *kv_list, size_t kv_count);
struct str *lumberjack_encode_data(uint32_t sequence, const char *payload, size_t payload_len);

int lumberjack_connect(struct lumberjack *lumberjack);
int lumberjack_connected(struct lumberjack *lumberjack);
void lumberjack_disconnect(struct lumberjack *lumberjack);
int lumberjack_write(struct lumberjack *lumberjack, struct str *payload);
int lumberjack_write_window_size(struct lumberjack *lumberjack, uint32_t window_size);

int lumberjack_read_ack(struct lumberjack *lumberjack, uint32_t *sequence_ret);

#endif /* _PROTO_H_ */

#ifndef _PROTO_H_
#define _PROTO_H_
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <netdb.h>
#include "openssl/ssl.h"
#include "ring.h"
#include "str.h"

struct kv {
  char *key;
  size_t key_len;
  char *value;
  size_t value_len;
}; /* struct kv */

struct lumberjack {
  const char *host;
  unsigned short port;

  /* internal state you don't need to access normally */
  int connected; /* are we connected? */
  uint32_t sequence; /* the current data frame sequence number */
  int fd; /* the socket conection (used by ssl) */
  SSL *ssl; /* the ssl connection */
  SSL_CTX *ssl_context; /* ssl context */

  size_t ring_size; /* the size of the ring */
  struct ring *ring; /* the ring buffer of things needing acknowledgement */

  struct str *io_buffer;
  struct str *compression_buffer;
};

#define LUMBERJACK_VERSION_1 '1'
#define LUMBERJACK_DATA_FRAME 'D'
#define LUMBERJACK_ACK_FRAME 'A'
#define LUMBERJACK_WINDOW_SIZE_FRAME 'W'
#define LUMBERJACK_COMPRESSED_BLOCK_FRAME 'C'

/* Create a new lumberjack client.
 *
 * - host is a hostname or IP address.
 * - port is the port to connect to.
 * - window_size is how many events to send before waiting for an ack.
 *
 * If the hostname resolves to multiple addresses, one address is picked at
 * random each time a connection is made.
 */
struct lumberjack *lumberjack_new(const char *host, unsigned short port, size_t window_size);

/* Tell lumberjack about an SSL cert/ca it should trust 
 *
 * - path is a string; can be a path to a file or directory.
 */
int lumberjack_set_ssl_ca(struct lumberjack *lumberjack, const char *path);

/** PUBLIC API */
/* Send a data frame with a given payload and length */
int lumberjack_send_data(struct lumberjack *lumberjack, const char *payload,
                         size_t payload_len);
                         //void (*free_func)(void *payload, void *hint()));

int lumberjack_flush(struct lumberjack *lumberjack);
/* TODO(sissel): permit inspection of currently-unacknowledged events? */

//int lumberjack_send_kv(struct *kv map);

/* blocks until all messages in the ring have been acknowledged */
void lumberjack_disconnect(struct lumberjack *lumberjack);
int lumberjack_ensure_connected(struct lumberjack *lumberjack);

/* Pack a key-value list according to the lumberjack protocol */
struct str *lumberjack_kv_pack(struct kv *kv_list, size_t kv_count);

//struct str *lumberjack_encode_data(uint32_t sequence, const char *payload, size_t payload_len);
//int lumberjack_connect(struct lumberjack *lumberjack);
//int lumberjack_connected(struct lumberjack *lumberjack);
//void lumberjack_disconnect(struct lumberjack *lumberjack);
//int lumberjack_write(struct lumberjack *lumberjack, struct str *payload);
//int lumberjack_read_ack(struct lumberjack *lumberjack, uint32_t *sequence_ret);

#endif /* _PROTO_H_ */

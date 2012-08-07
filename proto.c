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
#include "backoff.h"
#include "insist.h"

#include <openssl/bio.h>
#include <openssl/ssl.h>
#include <openssl/err.h>

static struct timespec MIN_SLEEP = { 0, 10000000 }; /* 10ms */
static struct timespec MAX_SLEEP = { 15, 0 }; /* 15 */

static void lumberjack_init(void);
static int lumberjack_tcp_connect(struct lumberjack *lumberjack);
static int lumberjack_ssl_handshake(struct lumberjack *lumberjack);
static int lumberjack_connected(struct lumberjack *lumberjack);
static int lumberjack_wait_for_ack(struct lumberjack *lumberjack);
static int lumberjack_ensure_connected(struct lumberjack *lumberjack);
static int lumberjack_retransmit_all(struct lumberjack *lumberjack);

static int lumberjack_init_done = 0;

static void lumberjack_init(void) {
  if (lumberjack_init_done) {
    return;
  }

  /* Seed the RNG so we can pick a random starting sequence number */
  srand(time(NULL));

  /* ssl init */
  CRYPTO_malloc_init();
  SSL_library_init();
  SSL_load_error_strings();
  ERR_load_BIO_strings();
  OpenSSL_add_all_algorithms();
  lumberjack_init_done = 1;
} /* lumberjack_init */

struct lumberjack *lumberjack_new(const char *host, unsigned short port) {
  struct lumberjack *lumberjack;
  lumberjack_init(); /* global one-time init */

  lumberjack = malloc(sizeof(struct lumberjack));
  lumberjack->host = host;
  lumberjack->port = port;
  lumberjack->fd = -1;
  lumberjack->sequence = rand();
  lumberjack->ssl = NULL;
  lumberjack->ring = ring_new_size(2048);
  return lumberjack;
} /* lumberjack_new */

int lumberjack_connect(struct lumberjack *lumberjack) {
  /* TODO(sissel): support ipv6, if anyone ever uses that in production ;) */
  insist(lumberjack != NULL, "lumberjack must not be NULL");
  insist(lumberjack->fd < 0, "already connected (fd %d > 0)", lumberjack->fd);
  insist(lumberjack->host != NULL, "lumberjack host must not be NULL");
  insist(lumberjack->port > 0, "lumberjack port must be > 9 (is %hd)", lumberjack->port);

  int rc;
  rc = lumberjack_tcp_connect(lumberjack);
  if (rc < 0) {
    return -1;
  }

  rc = lumberjack_ssl_handshake(lumberjack);
  if (rc < 0) {
    return -1;
  }

  /* If we get here, tcp connect + ssl handshake has succeeded */
  lumberjack->connected = 1;

  /* Retransmit anything currently in the ring (unacknowledged data frames) 
   * This is a no-op if there's nothing in the ring. */
  rc = lumberjack_retransmit_all(lumberjack);
  if (rc < 0) {
    /* Retransmit failed, which means a write failed during retransmit,
     * disconnect and claim a connection failure. */
    lumberjack_disconnect(lumberjack);
    return -1;
  }
  return 0;
} /* lumberjack_connect */

static int lumberjack_ensure_connected(struct lumberjack *lumberjack) {
  int rc;
  struct backoff sleeper;
  backoff_init(&sleeper, &MIN_SLEEP, &MAX_SLEEP);

  while (!lumberjack_connected(lumberjack)) {
    backoff(&sleeper);
    rc = lumberjack_connect(lumberjack);
    if (rc != 0) {
      printf("Connection attempt to %s:%hd failed: %s\n",
             lumberjack->host, lumberjack->port, strerror(errno));
    } else {
      /* we're connected! */
      backoff_clear(&sleeper);
    }
  }
  return 0;
} /* lumberjack_connect_block */

/* Connect to a host:port. If 'host' resolves to multiple addresses, one is
 * picked at random. */
static int lumberjack_tcp_connect(struct lumberjack *lumberjack) {
  int rc;
  int fd;
  struct hostent *hostinfo = gethostbyname(lumberjack->host);

  if (hostinfo == NULL) {
    /* DNS error, gethostbyname sets h_errno on failure */
    printf("gethostbyname(%s) failed: %s\n", lumberjack->host,
           strerror(h_errno));
    return -1;
  }

  /* 'struct hostent' has the list of addresses resolved in 'h_addr_list'
   * It's a null-terminated list, so count how many are there. */
  unsigned int addr_count;
  for (addr_count = 0; hostinfo->h_addr_list[addr_count] != NULL; addr_count++);
  /* hostnames can resolve to multiple addresses, pick one at random. */
  char *address = hostinfo->h_addr_list[rand() % addr_count];

  printf("Connecting to %s(%s):%hd\n", lumberjack->host,
         inet_ntoa(*(struct in_addr *)address), lumberjack->port);
  fd = socket(PF_INET, SOCK_STREAM, 0);
  insist(fd >= 0, "socket() failed: %s\n", strerror(errno));

  struct sockaddr_in sockaddr;
  sockaddr.sin_family = PF_INET,
  sockaddr.sin_port = htons(lumberjack->port),
  memcpy(&sockaddr.sin_addr, address, hostinfo->h_length);

  rc = connect(fd, (struct sockaddr *)&sockaddr, sizeof(sockaddr));
  if (rc < 0) {
    return -1;
  }

  printf("Connected successfully to %s(%s):%hd\n", lumberjack->host,
         inet_ntoa(*(struct in_addr *)address), lumberjack->port);

  lumberjack->fd = fd;
  return 0;
} /* lumberjack_tcp_connect */

static int lumberjack_ssl_handshake(struct lumberjack *lumberjack) {
  int rc;
  SSL_CTX *ctx = SSL_CTX_new(SSLv23_client_method());

  BIO *bio = BIO_new_socket(lumberjack->fd, 0 /* don't close on free */);
  if (bio == NULL) {
    ERR_print_errors_fp(stderr);
    insist(bio != NULL, "BIO_new_socket failed");
  }

  lumberjack->ssl = SSL_new(ctx);
  SSL_set_connect_state(lumberjack->ssl); /* we're a client */
  SSL_set_mode(lumberjack->ssl, SSL_MODE_AUTO_RETRY); /* retry writes/reads that would block */
  SSL_set_bio(lumberjack->ssl, bio, bio);

  struct backoff sleeper;
  backoff_init(&sleeper, &MIN_SLEEP, &MAX_SLEEP);
  for (rc = SSL_connect(lumberjack->ssl); rc < 0; rc = SSL_connect(lumberjack->ssl)) { 
    /* loop until ssl handshake succeeds */
    switch(SSL_get_error(lumberjack->ssl, rc)) {
      case SSL_ERROR_WANT_READ:
      case SSL_ERROR_WANT_WRITE:
        backoff(&sleeper);
        continue; /* retry */
      default:
        /* Some other SSL error */
        BIO_free_all(bio);
        lumberjack_disconnect(lumberjack);
        ERR_print_errors_fp(stderr);
        return -1;
    }
  }
  return 0;
} /* lumberjack_ssl_handshake */

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
    //bytes = write(lumberjack->fd, str_data(payload) + offset, remaining);
    bytes = SSL_write(lumberjack->ssl, str_data(payload) + offset, remaining);
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
  if (lumberjack->connected) {
    printf("Disconnect requested\n");
    if (lumberjack->ssl) {
      SSL_shutdown(lumberjack->ssl);
      lumberjack->ssl = NULL;
    }
    if (lumberjack->fd >= 0) {
      close(lumberjack->fd);
      lumberjack->fd = -1;
    }

    lumberjack->connected = 0;
    insist(!lumberjack_connected(lumberjack), 
           "lumberjack_connected() must not return true after a disconnect");
  }
} /* lumberjack_disconnect */
  
int lumberjack_connected(struct lumberjack *lumberjack) {
  return lumberjack->connected;
} /* lumberjack_connected */

static int lumberjack_read_ack(struct lumberjack *lumberjack, uint32_t *sequence_ret) {
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
    bytes = SSL_read(lumberjack->ssl, buf + offset, remaining);
    if (bytes <= 0) {
      /* error(<0) or EOF(0) */
      printf("bytes <= 0: %ld\n", bytes);
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

int lumberjack_send_data(struct lumberjack *lumberjack, const char *payload,
                         size_t payload_len) {
  /* TODO(sissel): support a 'free' function to free the payload when it's done */
                         //void (*free_func)(void *payload, void *hint()))
  struct str *frame = str_new_size(sizeof(uint32_t) + payload_len);
  int rc;

  lumberjack->sequence++;
  /* TODO(sissel): How to handle sequence value overflow (MAX_INT -> 0) */

  str_append_char(frame, LUMBERJACK_VERSION_1);
  str_append_char(frame, LUMBERJACK_DATA_FRAME);
  str_append_uint32(frame, lumberjack->sequence);
  str_append(frame, payload, payload_len);

  lumberjack_ensure_connected(lumberjack);
  /* if the ring is currently full, we need to wait for acks. */
  while (ring_is_full(lumberjack->ring)) {
    /* read at least one ACK */
    lumberjack_wait_for_ack(lumberjack);
  }

  /* Send this data frame on the wire */
  rc = lumberjack_write(lumberjack, frame);
  if (rc < 0) {
    /* write failure, reconnect (which will resend) and such */
    lumberjack_disconnect(lumberjack);
    lumberjack_ensure_connected(lumberjack);
  }

  /* Push this into the ring buffer, indicating it needs to be acknowledged */
  rc = ring_push(lumberjack->ring, frame);
  insist(rc == RING_OK, "ring_push failed (returned %d, expected RING_OK(%d)",
         rc, RING_OK);

  return 0; /* SUCCESS */
} /* lumberjack_send_data */

static int lumberjack_wait_for_ack(struct lumberjack *lumberjack) {
  uint32_t ack;
  int rc;
  struct backoff sleeper;
  backoff_init(&sleeper, &MIN_SLEEP, &MAX_SLEEP);

  while ((rc = lumberjack_read_ack(lumberjack, &ack)) < 0) {
    /* read error. */
    printf("lumberjack_read_ack failed: %s\n", strerror(errno));
    lumberjack_disconnect(lumberjack);
    backoff(&sleeper);
    lumberjack_ensure_connected(lumberjack);
  }

  /* TODO(sissel): Verify this is even a sane ack */

  /* Acknowledge anything in the ring that has a sequence number <= this ack */
  /* Clear anything in the ring with a sequence less than the one just acked */
  for (int i = 0, count = ring_count(lumberjack->ring); i < count; i++) {
    struct str *frame;
    uint32_t cur_seq;

    /* look at, but don't remove, the first item in the ring */
    ring_peek(lumberjack->ring, 0, (void **)&frame);

    /* this is a silly way, but since the ring only stores strings right now */
    memcpy(&cur_seq, str_data(frame) + 2, sizeof(uint32_t));
    cur_seq = ntohl(cur_seq);

    if (cur_seq <= ack) {
      //printf("bulk ack: %d\n", cur_seq);
      ring_pop(lumberjack->ring, NULL); /* destroy this item */
      str_free(frame);
    } else {
      /* found a sequence number > this ack,
       * we're done purging acknowledgements */
    }
  }

  return 0;
} /* lumberjack_wait_for_ack */

static int lumberjack_retransmit_all(struct lumberjack *lumberjack) {
  int rc;
  /* New connection, anything in the ring buffer is assumed to be
   * un-acknowledged. Send it. */
  for (int i = 0, count = ring_count(lumberjack->ring); i < count; i++) {
    struct str *frame;
    //uint32_t seq;
    rc = ring_peek(lumberjack->ring, i, (void **)&frame);
    insist(rc == RING_OK, "ring_peek(%d) failed unexpectedly: %d\n", i, rc);
    //memcpy(&seq, str_data(frame) + 2, sizeof(uint32_t));
    //seq = ntohl(seq);
    //printf("Retransmitting seq %d\n", seq);
    rc = lumberjack_write(lumberjack, frame);

    if (rc != 0) {
      /* write failure, fail. */
      return -1;
    }
  } /* for each item in the ring */

  return 0;
} /* lumberjack_retransmit_all */

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

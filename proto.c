#define _BSD_SOURCE /* for hstrerror */
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
#include <sys/stat.h>
#include <netdb.h>

#include "zlib.h"
#include "backoff.h"
#include "insist.h"
#include "openssl/bio.h"
#include "openssl/ssl.h"
#include "openssl/err.h"
#include "openssl/rand.h"

#include "strlist.h"
#include "sleepdefs.h"
#include "flog.h"

#define LUMBERJACK_POLL_READ 0x01
#define LUMBERJACK_POLL_WRITE 0x02

static void lumberjack_init(void);
static int lumberjack_tcp_connect(struct lumberjack *lumberjack);
static int lumberjack_ssl_handshake(struct lumberjack *lumberjack);
static int lumberjack_connected(struct lumberjack *lumberjack);
static int lumberjack_wait_for_ack(struct lumberjack *lumberjack);
static int lumberjack_retransmit_all(struct lumberjack *lumberjack);
static int lumberjack_write_window_size(struct lumberjack *lumberjack);
static int lumberjack_poll(struct lumberjack *lumberjack,
                           time_t seconds, int flags);

static int lumberjack_init_done = 0;

static unsigned int rand_uint32() {
  unsigned char bytes[4];
  /* TODO(sissel): Check error code? -1 means we failed... */
  RAND_pseudo_bytes(bytes, 4);
  return *(unsigned int *)(bytes);
} /* rand_uint32 */

static void lumberjack_init(void) {
  if (lumberjack_init_done) {
    return;
  }

  /* ssl init */
  CRYPTO_malloc_init();
  SSL_library_init();
  SSL_load_error_strings();
  ERR_load_BIO_strings();
  OpenSSL_add_all_algorithms();
  lumberjack_init_done = 1;
} /* lumberjack_init */

struct lumberjack *lumberjack_new(const char *host, unsigned short port, size_t window_size) {
  struct lumberjack *lumberjack;
  lumberjack_init(); /* global one-time init */

  lumberjack = malloc(sizeof(struct lumberjack));
  lumberjack->host = host;
  lumberjack->port = port;
  lumberjack->fd = -1;
  lumberjack->sequence = 0; //rand();
  lumberjack->ssl = NULL;
  lumberjack->connected = 0;

  /* I tried with 128, 256, 512, 1024, 2048, and 16384,
   * in a local network, an window size of 1024 seemed to have the best
   * performance (equal to 2048 and 16384) for the least memory cost. */
  if (window_size < 1024) {
    flog(stdout, "Window size less than 1024 (%d) isn't shown to have " \
         "speed-performance benefits", window_size);
  }

  lumberjack->ring_size = window_size;
  lumberjack->ring = ring_new_size(lumberjack->ring_size);

  /* Create this once. */
  lumberjack->ssl_context = SSL_CTX_new(SSLv23_client_method());
  SSL_CTX_set_verify(lumberjack->ssl_context, SSL_VERIFY_PEER, NULL);

  lumberjack->io_buffer = str_new_size(16384); /* TODO(sissel): tunable */

  /* zlib provides compressBound() to give a 'worst case' compressed
   * payload size on a input payload of a given size. */
  lumberjack->compression_buffer = str_new_size(compressBound(16384));
  return lumberjack;
} /* lumberjack_new */

int lumberjack_set_ssl_ca(struct lumberjack *lumberjack, const char *path) {
  insist(lumberjack != NULL, "lumberjack is NULL");
  insist(path != NULL, "path is NULL");
  insist(!lumberjack_connected(lumberjack),
         "You cannot call lumberjack_set_ssl_ca while connected.");

  int rc;
  /* Check whether 'path' is a directory or not. */
  struct stat path_stat;
  rc = stat(path, &path_stat);
  if (rc == -1) {
    /* Failed to stat the file */
    flog(stdout, "lumberjack_set_ssl_ca: stat(%s) failed: %s",
         path, strerror(errno));
    return -1;
  }

  if (S_ISDIR(path_stat.st_mode)) {
    rc = SSL_CTX_load_verify_locations(lumberjack->ssl_context, NULL, path);
  } else {
    /* assume a file */
    rc = SSL_CTX_load_verify_locations(lumberjack->ssl_context, path, NULL);
  }

  if (rc == 0) {
    ERR_print_errors_fp(stdout);
    return -1;
  }

  return 0;
} /* lumberjack_set_ssl_ca */

int lumberjack_connect(struct lumberjack *lumberjack) {
  /* TODO(sissel): support ipv6, if anyone ever uses that in production ;) */
  insist(lumberjack != NULL, "lumberjack must not be NULL");

  int rc;
  rc = lumberjack_tcp_connect(lumberjack);
  if (rc < 0) {
    return -1;
  }

  rc = lumberjack_ssl_handshake(lumberjack);
  if (rc < 0) {
    flog(stdout, "ssl handshake failed");
    lumberjack_disconnect(lumberjack);
    return -1;
  }

  /* If we get here, tcp connect + ssl handshake has succeeded */
  lumberjack->connected = 1;

  /* Always truncate the output buffer on a new connection.
   * This prevents accidental buffer leaks from an old/dead connection 
   * into this new one. 
   *
   * The symptom of such a leak is that, upon a new connection, 
   * the oldest non-ack'd event is not the first event seen by the receiver. 
   */
  str_truncate(lumberjack->io_buffer);

  /* Send our window size */
  rc = lumberjack_write_window_size(lumberjack);
  if (rc < 0) {
    flog(stdout, "lumberjack_write_window_size failed: %d", rc);
    lumberjack_disconnect(lumberjack);
    return -1;
  }

  /* Retransmit anything currently in the ring (unacknowledged data frames) 
   * This is a no-op if there's nothing in the ring. */
  rc = lumberjack_retransmit_all(lumberjack);
  if (rc < 0) {
    flog(stdout, "lumberjack_retransmit_all failed");
    /* Retransmit failed, which means a write failed during retransmit,
     * disconnect and claim a connection failure. */
    lumberjack_disconnect(lumberjack);
    return -1;
  }

  insist(lumberjack->fd > 0,
         "lumberjack->fd must be > 0 after a connect, was %d", lumberjack->fd);
  insist(lumberjack->ssl != NULL,
         "lumberjack->ssl must not be NULL after a connect");
  return 0;
} /* lumberjack_connect */

int lumberjack_ensure_connected(struct lumberjack *lumberjack) {
  int rc;
  struct backoff sleeper;
  backoff_init(&sleeper, &MIN_SLEEP, &MAX_SLEEP);

  while (!lumberjack_connected(lumberjack)) {
    backoff(&sleeper);
    rc = lumberjack_connect(lumberjack);
    if (rc != 0) {
      flog(stdout, "Connection attempt to %s:%hd failed: %s",
           lumberjack->host, lumberjack->port, strerror(errno));
    } else {
      /* we're connected! */
      backoff_clear(&sleeper);
    }
  }
  insist(lumberjack->fd > 0,
         "lumberjack->fd must be > 0 after a connect, was %d", lumberjack->fd);
  insist(lumberjack->ssl != NULL,
         "lumberjack->ssl must not be NULL after a connect");
  return 0;
} /* lumberjack_connect_block */

static struct hostent *lumberjack_choose_address(const char *host) {
  strlist_t *hostlist;
  split(&hostlist, host, ",");
  insist(hostlist->nitems > 0, "host string must not be empty");

  struct backoff sleeper;
  backoff_init(&sleeper, &MIN_SLEEP, &MAX_SLEEP);

  struct hostent *hostinfo = NULL;
  while (hostinfo == NULL) {
    int item = rand_uint32() % hostlist->nitems;
    char *chosen = hostlist->items[item];
    flog_if_slow(stdout, 0.200, {
      hostinfo = gethostbyname(chosen);
    }, "gethostbyname('%s')", chosen);
    if (hostinfo == NULL) {
      flog(stdout, "gethostbyname(%s) failed: %s", chosen,
           hstrerror(h_errno));
      backoff(&sleeper);
    } else {
      if (hostlist->nitems > 1) {
        /* only log that we've selected a random host if there's multiple hosts
         * in the list. */
        flog(stdout, "Random host selection: %s from %s", chosen, host);
      }
    }
  }
  strlist_free(hostlist);
  return hostinfo;
} /* lumberjack_choose_address */

/* Connect to a host:port. If 'host' resolves to multiple addresses, one is
 * picked at random. */
static int lumberjack_tcp_connect(struct lumberjack *lumberjack) {
  insist(lumberjack->fd < 0, "already connected (fd %d > 0)", lumberjack->fd);
  insist(lumberjack->host != NULL, "lumberjack host must not be NULL");
  insist(lumberjack->port > 0, "lumberjack port must be > 9 (is %hd)", lumberjack->port);
  insist(lumberjack != NULL, "lumberjack must not be NULL");
  int rc;
  int fd;

  struct hostent *hostinfo = lumberjack_choose_address(lumberjack->host);

  /* 'struct hostent' has the list of addresses resolved in 'h_addr_list'
   * It's a null-terminated list, so count how many are there. */
  unsigned int addr_count;
  for (addr_count = 0; hostinfo->h_addr_list[addr_count] != NULL; addr_count++);
  /* hostnames can resolve to multiple addresses, pick one at random. */
  char *address = hostinfo->h_addr_list[rand_uint32() % addr_count];

  flog(stdout, "Connecting to %s(%s):%hu", hostinfo->h_name,
         inet_ntoa(*(struct in_addr *)address), lumberjack->port);
  fd = socket(PF_INET, SOCK_STREAM, 0);
  insist(fd >= 0, "socket() failed: %s", strerror(errno));

  struct sockaddr_in sockaddr;
  sockaddr.sin_family = PF_INET;
  sockaddr.sin_port = htons(lumberjack->port);
  memcpy(&sockaddr.sin_addr, address, hostinfo->h_length);

  flog_if_slow(stdout, 0.250, {
    rc = connect(fd, (struct sockaddr *)&sockaddr, sizeof(sockaddr));
  }, "connect to %s:%hu", inet_ntoa(*(struct in_addr *)address), lumberjack->port);
  if (rc < 0) {
    close(fd);
    return -1;
  }

  flog(stdout, "Connected successfully to %s(%s):%hu", hostinfo->h_name,
       inet_ntoa(*(struct in_addr *)address), lumberjack->port);

  lumberjack->fd = fd;

  return 0;
} /* lumberjack_tcp_connect */

static int lumberjack_ssl_handshake(struct lumberjack *lumberjack) {
  insist(lumberjack != NULL, "lumberjack must not be NULL");
  insist(lumberjack->ssl == NULL, "ssl already established, cannot handshake");

  int rc;
  BIO *bio = BIO_new_socket(lumberjack->fd, 0 /* don't close on free */);
  if (bio == NULL) {
    ERR_print_errors_fp(stdout);
    insist(bio != NULL, "BIO_new_socket failed");
  }

  lumberjack->ssl = SSL_new(lumberjack->ssl_context);
  insist(lumberjack->ssl != NULL, "SSL_new must not return NULL");

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
        /* TODO(sissel): Instead of sleeping, we should call select() for
         * whichever case (read or write) is wanted */
        backoff(&sleeper);
        continue; /* retry */
      default:
        /* Some other SSL error */
        flog(stdout, "SSL_connect error vv");
        ERR_print_errors_fp(stdout);
        flog(stdout, "SSL_connect error ^^");
        return -1;
    }
  }
  ERR_print_errors_fp(stdout);

  /* TODO(sissel): Verify peer certificate */
  return 0;
} /* lumberjack_ssl_handshake */

int lumberjack_write(struct lumberjack *lumberjack, struct str *payload) {
  insist(lumberjack != NULL, "lumberjack must not be NULL");
  insist(payload != NULL, "payload must not be NULL");
  insist(lumberjack->ssl != NULL, "lumberjack->ssl must not be NULL");

  /* writing is an error if you are not connected */
  if (!lumberjack_connected(lumberjack)) {
    return -1;
  }

  /* TODO(sissel): For compression
   * - append payload to the buffer
   * - if the buffer is longer than BUFFER_FLUSH_SIZE
   *   - lumberjack_flush()
   * - else continue, do not write on the wire.
   */
  str_append_str(lumberjack->io_buffer, payload);

  //uint32_t seq;
  //memcpy(&seq, str_data(frame) + 2, sizeof(uint32_t));
  //seq = ntohl(seq);
  //flog(stdout, "io_buffer: %.*s", str_length(lumberjack->io_buffer), str_data(lumberjack->io_buffer));

  if (str_length(lumberjack->io_buffer) > 16384) {
    //flog(stdout, "io_buffer large enough (%d), flushing",
         //str_length(lumberjack->io_buffer));
    return lumberjack_flush(lumberjack);
  }
  return 0;
} /* lumberjack_write */

int lumberjack_flush(struct lumberjack *lumberjack) {
  ssize_t bytes;
  size_t length = str_length(lumberjack->io_buffer);
  /* Zlib */
  int rc;

  if (length == 0) {
    return 0; /* nothing to do */
  }

  if (!lumberjack_connected(lumberjack)) {
    return -1;
  }

  uLongf compressed_length = (uLongf) lumberjack->compression_buffer->data_size;
  /* compress2 is provided by zlib */
  rc = compress2((Bytef *)str_data(lumberjack->compression_buffer), 
                 &compressed_length,
                 (Bytef *)str_data(lumberjack->io_buffer), length, 1);
  insist(rc == Z_OK, "compress2(..., %lu, ..., %zd) failed; returned %d",
         compressed_length, length, rc);

  str_truncate(lumberjack->io_buffer);
  //flog(stdout, "lumberjack_flush: flushing %d bytes (compressed to %d bytes)",
       //(int)length, (int)compressed_length);

  /* TODO(sissel): Handle timeouts on any writes. */
  /* Write the 'compressed block' frame header */
  struct str *header = str_new_size(6);
  str_append_char(header, LUMBERJACK_VERSION_1);
  str_append_char(header, LUMBERJACK_COMPRESSED_BLOCK_FRAME);
  str_append_uint32(header, compressed_length);

  time_t timeout = 30;
  rc = lumberjack_poll(lumberjack, timeout, LUMBERJACK_POLL_WRITE);
  if ((rc & LUMBERJACK_POLL_WRITE) == 0) {
    /* socket was not writable after the given timeout. Fail it. */
    flog(stdout, "Waited %d seconds for a writable socket. Giving up.", timeout);
    lumberjack_disconnect(lumberjack);
    return -1;
  }

  flog_if_slow(stdout, 1.0, {
    bytes = SSL_write(lumberjack->ssl, str_data(header), str_length(header));
  }, "SSL_write (lumberjack compressed header)", NULL);
  str_free(header);

  if (bytes < 0) {
    /* error occurred while writing. */
    rc = SSL_get_error(lumberjack->ssl, bytes);
    if (rc == SSL_ERROR_SYSCALL) {
      flog(stdout, "SSL_write failed: %s", strerror(errno));
    } else {
      flog(stdout, "SSL_write returned %d (code: %d), something is wrong",
           bytes, rc);
      flog(stdout, "SSL_write error vv");
      ERR_print_errors_fp(stdout);
      flog(stdout, "SSL_write error ^^");
    }
    lumberjack_disconnect(lumberjack);
    return -1;
  }

  /* write the compressed payload */
  ssize_t remaining = compressed_length;
  size_t offset = 0;
  
  struct timeval start;
  gettimeofday(&start, NULL);
  double elapsed;

  struct backoff sleeper;
  backoff_init(&sleeper, &MIN_SLEEP, &MAX_SLEEP);

  while (remaining > 0) {
    time_t timeout = 30;
    rc = lumberjack_poll(lumberjack, timeout, LUMBERJACK_POLL_WRITE);
    if ((rc & LUMBERJACK_POLL_WRITE) == 0) {
      /* socket was not writable after the given timeout. Fail it. */
      flog(stdout, "Waited %d seconds for a writable socket. Giving up.", timeout);
      lumberjack_disconnect(lumberjack);
      return -1;
    }

    bytes = SSL_write(lumberjack->ssl,
                      str_data(lumberjack->compression_buffer) + offset,
                      remaining);

    if (bytes < 0) {
      elapsed = duration(&start);
      if (elapsed > 30) {
        flog(stdout, "SSL_write took too long (%.3f seconds), assuming " \
             "dead/busy server and disconnecting.", elapsed);
        lumberjack_disconnect(lumberjack);
        return -1;
      }

      rc = SSL_get_error(lumberjack->ssl, bytes);
      if (rc == SSL_ERROR_WANT_READ) {
        /* TODO(sissel): instead of backing off, select for read, then retry
         * the write. */
        backoff(&sleeper);
      } else if (rc == SSL_ERROR_WANT_WRITE) {
        /* TODO(sissel): instead of backing off, select for write, then retry
         * the write. */
        backoff(&sleeper);
      }
    }

    remaining -= bytes;
    offset += bytes;
  }
  return 0;
} /* lumberjack_flush */

void lumberjack_disconnect(struct lumberjack *lumberjack) {
  flog(stdout, "Disconnect requested");
  if (lumberjack->ssl) {
    SSL_shutdown(lumberjack->ssl);
    SSL_free(lumberjack->ssl);
    lumberjack->ssl = NULL;
  }
  if (lumberjack->fd >= 0) {
    close(lumberjack->fd);
    lumberjack->fd = -1;
  }

  lumberjack->connected = 0;
  insist(!lumberjack_connected(lumberjack), 
         "lumberjack_connected() must not return true after a disconnect");
} /* lumberjack_disconnect */
  
int lumberjack_connected(struct lumberjack *lumberjack) {
  return lumberjack->connected;
} /* lumberjack_connected */

static int lumberjack_read_ack(struct lumberjack *lumberjack,
                               uint32_t *sequence_ret) {
  if (!lumberjack_connected(lumberjack)) {
    flog(stdout, "NOT CONNECTED");
    return -1;
  }

  /* This is a subpar implementation... reading 6 bytes at a time, etc, 
   * but the idea is that you can do bulk acks, so data-to-ack ratio should be
   * high */
  char buf[6]; 
  ssize_t bytes;
  size_t remaining = 6; /* version + frame type + 32bit sequence value */
  size_t offset = 0;

  int rc;
  while (remaining > 0) {
    /* Allow a few seconds for a read timeout. If it occurs, fail this read. 
     * This timeout should cause a disconnect and reconnect to a new server.
     * The idea is to prevent one receiving server from becoming overloaded. */
    time_t timeout = 30;
    rc = lumberjack_poll(lumberjack, timeout, LUMBERJACK_POLL_READ);
    if ((rc & LUMBERJACK_POLL_READ) == 0) {
      /* socket was not writable after the given timeout. Fail it. */
      flog(stdout, "Waited %d seconds for a readable socket. Giving up.", timeout);
      errno = ETIMEDOUT;
      return -1;
    }

    flog_if_slow(stdout, 1.0, {
      bytes = SSL_read(lumberjack->ssl, buf + offset, remaining);
    }, "SSL_read (tried to read %d bytes)", remaining);
    if (bytes == 0) {
      /* EOF or some other similar error */
      errno = EPIPE; /* close enough to fake EOF? */
      return -1;
    } else if (bytes < 0) {
      rc = SSL_get_error(lumberjack->ssl, bytes /* error code */);
      flog(stdout, "SSL_read error vv");
      ERR_print_errors_fp(stdout);
      flog(stdout, "SSL_read error ^^");
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
    /* flush any writes waiting for buffering/compression */
    flog(stdout, "ring buffer is full (%d items), flushing.",
         ring_count(lumberjack->ring));
    flog_if_slow(stdout, 0.500, {
      lumberjack_flush(lumberjack);
    }, "flush complete", NULL);

    /* read at least one ACK */
    flog_if_slow(stdout, 0.500, {
      rc = lumberjack_wait_for_ack(lumberjack);
      if (rc != 0) {
        flog(stdout, "lumberjack_wait_for_ack failed %d", rc);
      }
    }, "wait for ack (current sequence: %u)", lumberjack->sequence);
  }

  /* Send this data frame on the wire */
  rc = lumberjack_write(lumberjack, frame);
  if (rc != 0) {
    /* write failure, reconnect (which will resend) and such */
    flog(stdout, "lumberjack_write failed: %d", rc);
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

  flog(stdout, "lumberjack_wait_for_ack: waiting for ack");

  /* Wait for an ack */
  while ((rc = lumberjack_read_ack(lumberjack, &ack)) < 0) {
    /* read error. */
    flog(stdout, "lumberjack_read_ack failed: %s", strerror(errno));
    lumberjack_disconnect(lumberjack);
    backoff(&sleeper);
    lumberjack_ensure_connected(lumberjack);
  }

  flog(stdout, "lumberjack_wait_for_ack: received ack: %u", ack);

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
    rc = ring_peek(lumberjack->ring, i, (void **)&frame);
    insist(rc == RING_OK, "ring_peek(%d) failed unexpectedly: %d\n", i, rc);
    rc = lumberjack_write(lumberjack, frame);

    if (rc != 0) {
      /* write failure, fail. */
      flog(stdout, "write failure");
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

int lumberjack_write_window_size(struct lumberjack *lumberjack) {
  uint32_t size = lumberjack->ring_size;
  char data[6];

  flog(stdout, "Declaring window size of %u", size);

  data[0] = LUMBERJACK_VERSION_1;
  data[1] = LUMBERJACK_WINDOW_SIZE_FRAME;
  
  /* network byte order! */
  size = htonl(size);
  memcpy(data + 2, &size, sizeof(uint32_t));

  struct str payload = {
    .data_len = 6,
    .data_size = 6,
    .data = data
  };

  int rc;
  rc = lumberjack_write(lumberjack, &payload);
  if (rc != 0) {
    /* write failure, fail. */
    flog(stdout, "write failure while writing the window size");
    return -1;
  }
  return 0;
} /* lumberjack_write_window_size */

int lumberjack_poll(struct lumberjack *lumberjack, time_t seconds, int flags) {
  int ssl_fd = SSL_get_rfd(lumberjack->ssl);
  fd_set read_fds, write_fds;
  FD_ZERO(&read_fds);
  FD_ZERO(&write_fds);

  if (flags & LUMBERJACK_POLL_READ) {
    FD_SET(ssl_fd, &read_fds);
  }
  if (flags & LUMBERJACK_POLL_WRITE) {
    FD_SET(ssl_fd, &write_fds);
  }

  struct timeval timeout = { seconds, 0 }; 

  int rc = select(ssl_fd + 1 /* 'max fd in the list' */,
                    &read_fds, &write_fds, NULL, &timeout);
  if (rc == 0) {
    /* timeout, fail the read */
    errno = ETIMEDOUT;
    return 0;
  }

  int retval = 0;
  if (FD_ISSET(ssl_fd, &read_fds)) {
    retval |= LUMBERJACK_POLL_READ;
  }
  if (FD_ISSET(ssl_fd, &write_fds)) {
    retval |= LUMBERJACK_POLL_WRITE;
  }
  return retval;
} /* lumberjack_poll */

int lumberjack_ssl_write(const struct lumberjack *lumberjack,
                         const void *buffer, const int length,
                         const double timeout_seconds) {
  // Attempt to write 'length' bytes from buffer
  // If we can't write the full amount after 'timeout_seconds' passes, 
  // return ETIMEDOUT
  int rc;
  struct timeval start;
  gettimeofday(&start, NULL);
  double elapsed;
  int offset = 0;
  int remaining = length;

  struct backoff sleeper;
  backoff_init(&sleeper, &MIN_SLEEP, &MAX_SLEEP);

  //flog(stdout, "Writing [%d] %.*s", remaining, remaining, buffer + offset);
  while (remaining > 0) {
    rc = SSL_write(lumberjack->ssl, buffer + offset, remaining);

    if (rc < 0) {
      elapsed = duration(&start);
      if (elapsed > timeout_seconds) {
        flog(stdout, "SSL_write took too long (%.3f seconds), assuming " \
             "dead/busy server and disconnecting.", elapsed);
        errno = ETIMEDOUT;
        return -1;
      }

      int ssl_error = SSL_get_error(lumberjack->ssl, rc);
      if (ssl_error == SSL_ERROR_WANT_READ) {
        /* TODO(sissel): instead of backing off, select for read, then retry
         * the write. */
        backoff(&sleeper);
      } else if (ssl_error == SSL_ERROR_WANT_WRITE) {
        /* TODO(sissel): instead of backing off, select for write, then retry
         * the write. */
        backoff(&sleeper);
      } else {
        // Some other ssl error
        flog(stdout, "SSL_write failed: %d (ssl error: %d)", rc, ssl_error);
        return rc; // 
      }
    } else {
      remaining -= rc;
      offset += rc;
    }
  } /* while remaining > 0 */

  /* If we get here, success! */
  return 0;
} /* lumberjack_ssl_write */

int lumberjack_ssl_read(const struct lumberjack *lumberjack, void *buffer,
                        const int length, const double timeout_seconds) {
  // Attempt to read 'length' bytes from buffer
  // If we can't read the full amount after 'timeout_seconds' passes, 
  // return ETIMEDOUT
  int rc;
  struct timeval start;
  gettimeofday(&start, NULL);
  double elapsed;
  int offset = 0;
  int remaining = length;

  struct backoff sleeper;
  static struct timespec max_sleep = { 0, 250000000 }; /* 250ms */
  backoff_init(&sleeper, &MIN_SLEEP, &max_sleep);

  while (remaining > 0) {
    rc = SSL_read(lumberjack->ssl, buffer + offset, remaining);
    //flog(stdout, "SSL_read read %d ; returned %d", remaining, rc);

    if (rc > 0) {
      // Healthy read.
      remaining -= rc;
      offset += rc;
    } else if (rc == 0) {
      errno = EPIPE; // close enough to calling EOF as an error code
      return -1;
    } else if (rc < 0) {
      elapsed = duration(&start);
      if (elapsed > timeout_seconds) {
        flog(stdout, "SSL_read took too long (%.3f seconds), assuming " \
             "dead/busy server and disconnecting.", elapsed);
        errno = ETIMEDOUT;
        return -1;
      }

      int ssl_error = SSL_get_error(lumberjack->ssl, rc);
      if (ssl_error == SSL_ERROR_WANT_READ) {
        /* TODO(sissel): instead of backing off, select for read, then retry
         * the write. */
        backoff(&sleeper);
      } else if (ssl_error == SSL_ERROR_WANT_WRITE) {
        /* TODO(sissel): instead of backing off, select for write, then retry
         * the write. */
        backoff(&sleeper);
      } else {
        // Some other ssl error
        flog(stdout, "SSL_read failed: %d (ssl error: %d)", rc, ssl_error);
        return rc; // 
      }
    }
  } /* while remaining > 0 */

  /* If we get here, success! */
  return 0;
} /* lumberjack_ssl_read */

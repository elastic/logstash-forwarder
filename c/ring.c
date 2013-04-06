#include "ring.h"
#include "insist.h"
#include <jemalloc/jemalloc.h>

struct ring *ring_new_size(uint32_t size) {
  insist((size & (size - 1)) == 0,
         "size must be a power of two, %d is not.", size);

  struct ring *r = malloc(sizeof(struct ring));
  r->writer = 0;
  r->reader = 0;
  r->count = 0;
  r->size = size;
  r->buffer = malloc(r->size * sizeof(void *));
  return r;
} /* ring_new_size */


inline int ring_is_empty(struct ring *ring) {
  return ring->count == 0;
} /* ring_is_empty */

inline int ring_is_full(struct ring *ring) {
  return ring->count == ring->size;
} /* ring_is_full */

inline int ring_peek(struct ring *ring, uint32_t i, void **object_ret) {
  if (i >= ring->count) {
    return RING_INDEX_OUT_OF_BOUNDS;
  }
  /* item 0 is the next one after the reader 
   * we mask with 'size - 1' as a way of wrapping the value since we enforce
   * power-of-two-ness */
  *object_ret = ring->buffer[(ring->reader + i) & (ring->size - 1)];
  return RING_OK;
} /* ring_peek */

inline int ring_pop(struct ring *ring, void **object_ret) {
  int rc;
  if (object_ret != NULL) {
    /* Only store it if object_ret is not NULL */
    rc = ring_peek(ring, 0, object_ret);
    if (rc != RING_OK) {
      return RING_IS_EMPTY;
    }
  }

  /* increment reader position and wrap write if necessary */
  ring->reader = (ring->reader + 1) & (ring->size - 1);
  ring->count--;
  return RING_OK;
} /* ring_pop */

inline int ring_push(struct ring *ring, void *object) {
  if (ring_is_full(ring)) {
    return RING_IS_FULL;
  }

  ring->buffer[ring->writer] = object;
  /* increment write position and wrap write if necessary */
  ring->writer = (ring->writer + 1) & (ring->size - 1);
  ring->count++;
  return RING_OK;
} /* ring_push */

inline uint32_t ring_count(struct ring *ring) {
  return ring->count;
} /* ring count */

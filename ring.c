#include "ring.h"
#include <jemalloc/jemalloc.h>

struct ring *ring_new_size(size_t size) {
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

int ring_is_full(struct ring *ring) {
  return ring->count == ring->size;
} /* ring_is_full */

int ring_pop(struct ring *ring, void **object_ret) {
  if (ring_is_empty(ring)) {
    return RING_IS_EMPTY;
  }

  *object_ret = ring->buffer[ring->reader];

  ring->reader++;
  ring->count--;
  if (ring->reader == ring->size) {
    ring->reader = 0; /* wrap around */
  }
  return RING_OK;
} /* ring_pop */

int ring_push(struct ring *ring, void *object) {
  if (ring_is_full(ring)) {
    return RING_IS_FULL;
  }

  ring->buffer[ring->writer] = object;
  ring->writer++;
  ring->count++;
  if (ring->writer == ring->size) {
    ring->writer = 0; /* wrap around */
  }
  return RING_OK;
} /* ring_push */

#include "ring.h"
#include <jemalloc/jemalloc.h>

struct ring *ring_new_size(size_t count, size_t object_size);
  struct ring *r;
  r = malloc(sizeof(struct ring));
  r->head = 0;
  r->tail = 0;
  r->size = size;
  r->buffer = malloc(count * object_size);
  return r;
} /* ring_new_size */

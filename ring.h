#ifndef _RING_H_
#define _RING_H_
#include <stdint.h>

struct ring {
  uint32_t writer; /* write position */
  uint32_t reader; /* read position */
  uint32_t size; /* maximum number of items */
  uint32_t count; /* current count of items */
  void **buffer; /* array of pointers to whatever objects we're storing */
};

#define RING_OK 0x00
#define RING_IS_EMPTY 0x01
#define RING_IS_FULL 0x02
#define RING_INDEX_OUT_OF_BOUNDS 0x03

struct ring *ring_new_size(uint32_t count);

int ring_is_empty(struct ring *ring);
int ring_is_full(struct ring *ring);

uint32_t ring_count(struct ring *ring);

int ring_pop(struct ring *ring, void **object_ret);
int ring_peek(struct ring *ring, uint32_t index, void **object_ret);
int ring_push(struct ring *ring, void *object);

#endif /* _RING_H_ */

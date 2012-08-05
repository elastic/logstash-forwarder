#ifndef _RING_H_
#define _RING_H_
#include <sys/types.h>

struct ring {
  size_t writer;
  size_t reader;
  size_t size;
  size_t count;
  void **buffer; /* array of pointers to whatever objects we're storing */
};

#define RING_OK 0x00
#define RING_IS_EMPTY 0x01
#define RING_IS_FULL 0x02

struct ring *ring_new_size(size_t count);

int ring_is_empty(struct ring *ring);
int ring_is_full(struct ring *ring);

int ring_pop(struct ring *ring, void **object_ret);
int ring_push(struct ring *ring, void *object);

#endif /* _RING_H_ */

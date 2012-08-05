#include "ring.h"
#include <string.h>
#include "insist.h"

int main(void) {
  struct ring *ring;
  ring = ring_new_size(4);

  char *val;
  insist(ring_is_empty(ring), "A new ring must be empty");
  insist(ring_push(ring, "Hello world 1") == RING_OK, "Pushing 1 into a 4-slot ring must be OK");
  ring_peek(ring, 0, (void **)&val);
  insist(strcmp(val, "Hello world 1") == 0, "ring_peek(0) failed");
  insist(ring_push(ring, "Hello world 2") == RING_OK, "Pushing 2 into a 4-slot ring must be OK");
  ring_peek(ring, 1, (void **)&val);
  insist(strcmp(val, "Hello world 2") == 0, "ring_peek(1) failed");
  insist(ring_push(ring, "Hello world 3") == RING_OK, "Pushing 3 into a 4-slot ring must be OK");
  ring_peek(ring, 2, (void **)&val);
  insist(strcmp(val, "Hello world 3") == 0, "ring_peek(2) failed");
  insist(ring_push(ring, "Hello world 4") == RING_OK, "Pushing 4 into a 4-slot ring must be OK");
  ring_peek(ring, 3, (void **)&val);
  insist(strcmp(val, "Hello world 4") == 0, "ring_peek(3) failed");
  insist(ring_push(ring, "Hello world 5") == RING_IS_FULL, "Pushing 5 into a 4-slot ring must fail ");
  insist(ring_is_full(ring), "The ring must be full at this point");
  insist(!ring_is_empty(ring), "Ring must not be empty at this point");

  insist(ring_pop(ring, (void **)&val) == RING_OK, "Popping from a full ring must succeed");
  insist(strcmp(val, "Hello world 1") == 0, "Got the wrong string?");
  insist(ring_pop(ring, (void **)&val) == RING_OK, "Popping on a non-empty ring must succeed");
  insist(strcmp(val, "Hello world 2") == 0, "Got the wrong string?");
  insist(ring_pop(ring, (void **)&val) == RING_OK, "Popping on a non-empty ring must succeed");
  insist(strcmp(val, "Hello world 3") == 0, "Got the wrong string?");
  insist(ring_pop(ring, (void **)&val) == RING_OK, "Popping on a non-empty ring must succeed");
  insist(strcmp(val, "Hello world 4") == 0, "Got the wrong string?");
  insist(ring_pop(ring, (void **)&val) == RING_IS_EMPTY, "Pop on an empty ring must fail");
  insist(ring_is_empty(ring), "Ring must be empty at this point");
  insist(!ring_is_full(ring), "Ring must not be full at this point");

  printf("%s OK\n", __FILE__);
  return 0;
} /* main */

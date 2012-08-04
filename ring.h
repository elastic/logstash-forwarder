
struct fifo {
  size_t head;
  size_t tail;
  size_t size;
  void *buffer[]; /* array of pointers to whatever objects we're storing */
};

struct ring *ring_new_size(size_t count, size_t object_size);

int ring_is_empty(struct ring *ring);
int ring_is_full(struct ring *ring);

void *ring_pop(struct ring *ring);
void ring_push(struct ring *ring, void *object);

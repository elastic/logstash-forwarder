#ifndef _EMITTER_H_
#define _EMITTER_H_

struct emitter_config {
  void *zmq; /* zmq context */
  char *zmq_endpoint; /* inproc://whatever */
};

void *emitter(void *arg);
#endif /* _EMITTER_H_ */

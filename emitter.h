#ifndef _EMITTER_H_
#define _EMITTER_H_

struct emitter_config {
  void *zmq; /* zmq context */
  char *zmq_endpoint; /* inproc://whatever */
  char *ssl_ca_path; /* path to trusted ssl ca, can be a directory or a file */

  char *host;
  short port;
  short verify; 
};

void *emitter(void *arg);
#endif /* _EMITTER_H_ */

#define _BSD_SOURCE
#include <pthread.h>
#include <string.h>
#include <zmq.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

//#define ENDPOINT "tcp://127.0.0.1:12345"
#define ENDPOINT "inproc://asdf"

void free2(void *data, void __attribute__((unused)) *hint) {
  free(data);
}

void *pusher(void *zmq) {
  void *socket = zmq_socket(zmq, ZMQ_PUSH);
  int rc;
  int64_t hwm = 1;
  zmq_setsockopt(socket, ZMQ_HWM, &hwm, sizeof(hwm));
  
  while (rc = zmq_connect(socket, ENDPOINT), rc != 0) {
    printf("pusher waiting for connect to succeed...\n");
    sleep(1);
  }

  for (;;) {
    zmq_msg_t msg;
    zmq_msg_init_data(&msg, strdup("Hello World"), 12, free2, NULL);
    zmq_send(socket, &msg, 0);
    zmq_msg_close(&msg);
  }
}

void *puller(void *zmq) {
  void *socket = zmq_socket(zmq, ZMQ_PULL);
  int64_t hwm = 1;
  int rc;
  zmq_setsockopt(socket, ZMQ_HWM, &hwm, sizeof(hwm));
  rc = zmq_bind(socket, ENDPOINT);
  if (rc != 0) {
    printf("bind(%s) failed: %s\n", ENDPOINT, zmq_strerror(errno));
    abort();
  }

  for (;;) {
    zmq_msg_t msg;
    zmq_msg_init(&msg);
    zmq_recv(socket, &msg, 0);
    zmq_msg_close(&msg);
  }
}

int main(int argc, char **argv) {
  void *zmq = zmq_init(0); /* inproc only, no threads needed */

  if (argc != 2) {
    printf("Usage: %s <THREADCOUNT>\n", argv[0]);
    return 1;
  }

  int i = 0;
  int threads = atoi(argv[1]);
  
  pthread_t p;
  pthread_create(&p, NULL, puller, zmq);
  /* Create pusher threads, thread count comes from command args */
  for (i = 0; i < threads; i++) {
    pthread_t *pushthread = calloc(1, sizeof(pthread_t));
    pthread_create(pushthread, NULL, pusher, zmq);
  }
  pthread_join(p, NULL);
  return 0;
}

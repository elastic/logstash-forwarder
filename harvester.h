#ifndef _HARVESTER_H_
#define _HARVESTER_H_

struct harvest_config {
  char *path; /* the path to harvest */

  void *zmq; /* zmq context */
  char *zmq_endpoint; /* inproc://whatever */
};

void *harvest(void *arg);
#endif /* _HARVESTER_H_ */

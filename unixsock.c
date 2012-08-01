#include <sys/socket.h> /* for socket(2) */
#include <sys/un.h> /* for struct sockaddr_un */
#include <errno.h> /* for errno */
#include <string.h> /* for strerror(3) */
#include <insist.h> /* github/jordansissel/insist */
#include <unistd.h> /* for unlink(2) */

int main() {
  int r;
  int sock;
  sock = socket(PF_LOCAL, SOCK_DGRAM, 0);
  insist(sock != -1, "socket() failed: %s", strerror(errno));

  struct sockaddr_un addr;
  addr.sun_family = PF_LOCAL;
  strcpy(addr.sun_path, "/tmp/fancylog");
  //unlink(addr.sun_path);
  r = bind(sock, (struct sockaddr *)&addr,
           sizeof(addr.sun_family) + strlen(addr.sun_path) + 1);
  insist(r == 0, "bind(%s) failed: %s", addr.sun_path, strerror(errno));

  char buffer[65536];
  for (;;) {
    ssize_t bytes;
    bytes = recvfrom(sock, buffer, 65536, 0, NULL, NULL);
    insist(bytes > 0, "recvfrom() returned %d: %s", (int)bytes, strerror(errno));
    printf("Received: %.*s\n", (int)bytes, buffer);
  }
} /* main */

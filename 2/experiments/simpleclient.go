package main

import (
  "fmt"
  zmq "github.com/alecthomas/gozmq"
  "time"
)

func main() {
  context, _ := zmq.NewContext()
  socket, _ := context.NewSocket(zmq.REQ)
  defer context.Close()
  defer socket.Close()

  socket.Connect("tcp://localhost:5555")

  data := "Hello world"
  for i := 0; ; i++ {
    start := time.Now()
    socket.Send([]byte(data), 0)
    send_time := time.Since(start)

    start = time.Now()
    reply, _ := socket.Recv(0)
    recv_time := time.Since(start)

    if (i == 10000) {
      fmt.Printf("Received: %s\n", string(reply))
      fmt.Printf("Send: %d\n", send_time.Nanoseconds())
      fmt.Printf("Recv: %d\n", recv_time.Nanoseconds())
      i = 0
    }
  }
}

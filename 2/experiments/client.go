package main

import (
  "fmt"
  zmq "github.com/alecthomas/gozmq"
  proto "code.google.com/p/goprotobuf/proto"
  "time"
  "lumberjack"
)

func main() {
  context, _ := zmq.NewContext()
  socket, _ := context.NewSocket(zmq.REQ)
  defer context.Close()
  defer socket.Close()

  fmt.Printf("Connecting\n")
  socket.Connect("tcp://localhost:5555")

  f := &lumberjack.FileEvent{}
  f.Source = proto.String("/var/log/message")
  f.Offset = proto.Uint64(0)
  f.Line = proto.Uint64(0)
  f.Text = proto.String("hello world")

  for i := 0; ; i++ {
    start := time.Now()
    data, _ := proto.Marshal(f)
    marshal_time := time.Since(start)

    start = time.Now()
    socket.Send(data, 0)
    send_time := time.Since(start)

    start = time.Now()
    reply, _ := socket.Recv(0)
    recv_time := time.Since(start)

    start = time.Now()
    r := &lumberjack.FileEvent{}
    _ = proto.Unmarshal(reply, r)
    unmarshal_time := time.Since(start)
    if (i == 10000) {
      fmt.Printf("Received: %v\n", r)
      fmt.Printf("Marshal: %d\n", marshal_time.Nanoseconds())
      fmt.Printf("Send: %d\n", send_time.Nanoseconds())
      fmt.Printf("Recv: %d\n", recv_time.Nanoseconds())
      fmt.Printf("Unmarshal: %d\n", unmarshal_time.Nanoseconds())
      i = 0
    }
  }
}

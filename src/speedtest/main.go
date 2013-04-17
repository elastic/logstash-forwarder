package main

import (
  "log"
  zmq "github.com/alecthomas/gozmq"
  "compress/zlib"
  "time"
  "sodium"
  lumberjack "liblumberjack"
  "io"
  "bytes"
  "encoding/json"
  "fmt"
)

const SPOOLSIZE uint64 = 16384

func main() {
  //f, err := os.OpenFile("log.out", os.O_WRONLY | os.O_CREATE, 0644)
  //log.SetOutput(f)
  event_chan := make(chan *lumberjack.FileEvent, 16)
  publisher_chan := make(chan []*lumberjack.FileEvent, 5)
  registrar_chan := make(chan []*lumberjack.FileEvent, 5)

  endpoint := "tcp://127.0.0.1:47342"
  public, secret := sodium.CryptoBoxKeypair()

  go generator(event_chan)
  go lumberjack.Spool(event_chan, publisher_chan, SPOOLSIZE, 5 * time.Second)
  go lumberjack.Publish(publisher_chan, registrar_chan, []string{endpoint},
                        public, secret, 5 * time.Second)

  session := sodium.NewSession(public, secret)
  context, _ := zmq.NewContext()
  socket, _ := context.NewSocket(zmq.REP)
  socket.SetSockOptInt(zmq.LINGER, 0)
  err := socket.Bind(endpoint)
  if err != nil {
    log.Fatalf("Failed to bind to %s.\n", endpoint)
  }

  var buffer bytes.Buffer
  var decompressed bytes.Buffer
  tmp := make([]byte, 2048)
  count := 0
  start := time.Now()

  for count < 800000 {
    nonce, err := socket.Recv(0)
    if err != nil { panic(fmt.Sprintf("socket.Recv: %s\n", err)) }
    ciphertext, err := socket.Recv(0)
    if err != nil { panic(fmt.Sprintf("socket.Recv2: %s\n", err)) }

    count += int(SPOOLSIZE); socket.Send([]byte(""), 0); continue

    // Decrypt it
    plaintext := session.Open(nonce, ciphertext)


    buffer.Truncate(0)
    buffer.Write(plaintext)
    zr, _ := zlib.NewReader(&buffer)
    decompressed.Truncate(0)
    for { 
      n, err := zr.Read(tmp)
        if n > 0 {
          decompressed.Write(tmp[0:n])
        }
      if err == io.EOF {
        break
      }
    }
    zr.Close()

    var events []lumberjack.FileEvent
    err = json.Unmarshal(decompressed.Bytes(), &events)
    if err != nil { panic("JSON Unmarshal failed") }
    count += len(events)
  }

  log.Printf("%d @ %f/sec\n", count, float64(count) / time.Since(start).Seconds())
}

func generator(output chan *lumberjack.FileEvent) {
  source := "whatever"
  var offset uint64 = 0
  var line uint64 = 0
  text := "hello world a b c def ghalskdjfl awkejtlk ajwet"
  for {
    event := lumberjack.FileEvent {
      Source: &source,
      Offset: offset,
      Line: line,
      Text: &text,
    }

    offset += uint64(len(text))
    line++

    output <- &event
  }
}

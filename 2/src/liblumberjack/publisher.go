package liblumberjack

import (
  "log"
  msgpack "github.com/ugorji/go-msgpack"
  zmq "github.com/alecthomas/gozmq"
  "math/big"
  "syscall"
  "bytes"
  "time"
  //"syscall"
  "compress/zlib"
  //"compress/flate"
  "crypto/rand"
)

var context zmq.Context
func init() {
  context, _ = zmq.NewContext()
}

// Forever Faithful Socket
type FFS struct {
  Endpoints []string // set of endpoints available to ship to

  // Socket type; zmq.REQ, etc
  SocketType zmq.SocketType

  // Various timeout values
  SendTimeout time.Duration
  RecvTimeout time.Duration

  endpoint string // the current endpoint in use
  socket zmq.Socket // the current zmq socket
  connected bool // are we connected?
}

func (s *FFS) Send(data []byte, flags zmq.SendRecvOption) (err error) {
  for {
    s.ensure_connect()

    pi := zmq.PollItems{zmq.PollItem{Socket: s.socket, Events: zmq.POLLOUT}}
    count, err := zmq.Poll(pi, int64(s.SendTimeout.Nanoseconds() / 1000))
    if count == 0 {
      // not ready in time, fail the socket and try again.
      log.Printf("%s: timed out waiting to Send(): %s\n",
                 s.endpoint, err)
      s.fail_socket()
    } else {
      //log.Printf("%s: sending %d payload\n", s.endpoint, len(data))
      err = s.socket.Send(data, flags)
      if err != nil {
        log.Printf("%s: Failed to Send() %d byte message: %s\n",
                   s.endpoint, len(data), err)
        s.fail_socket()
      } else {
        // Success!
        break
      }
    }
  }
  return
}

func (s *FFS) Recv(flags zmq.SendRecvOption) (data []byte, err error) {
  s.ensure_connect()

  pi := zmq.PollItems{zmq.PollItem{Socket: s.socket, Events: zmq.POLLIN}}
  count, err := zmq.Poll(pi, int64(s.RecvTimeout.Nanoseconds() / 1000))
  if count == 0 {
    // not ready in time, fail the socket and try again.
    s.fail_socket()

    err = syscall.ETIMEDOUT
    log.Printf("%s: timed out waiting to Recv(): %s\n",
               s.endpoint, err)
    return nil, err
  } else {
    data, err = s.socket.Recv(flags)
    if err != nil {
      log.Printf("%s: Failed to Recv() %d byte message: %s\n",
                 s.endpoint, len(data), err)
      s.fail_socket()
      return nil, err
    } else {
      // Success!
    }
  }
  return
}

func (s *FFS) Close() (err error) {
  err = s.socket.Close()
  if err != nil { return }

  s.socket = nil
  s.connected = false
  return nil
}

func (s *FFS) ensure_connect() {
  if s.connected {
    return
  }

  if s.SendTimeout == 0 {
    s.SendTimeout = 1 * time.Second
  }
  if s.RecvTimeout == 0 {
    s.RecvTimeout = 1 * time.Second
  }

  if s.SocketType == 0 {
    log.Panicf("No socket type set on zmq socket")
  }
  if s.socket != nil { 
    s.socket.Close()
    s.socket = nil
  }

  var err error
  s.socket, err = context.NewSocket(s.SocketType) 
  if err != nil {
    log.Panicf("zmq.NewSocket(%d) failed: %s\n", s.SocketType, err)
  }

  //s.socket.SetSockOptUInt64(zmq.HWM, 1)
  //s.socket.SetSockOptInt(zmq.RCVTIMEO, int(s.RecvTimeout.Nanoseconds() / 1000000))
  //s.socket.SetSockOptInt(zmq.SNDTIMEO, int(s.SendTimeout.Nanoseconds() / 1000000))

  // Abort anything in-flight on a socket that's closed.
  s.socket.SetSockOptInt(zmq.LINGER, 0)

  for !s.connected {
    var max *big.Int = big.NewInt(int64(len(s.Endpoints)))
    i, _ := rand.Int(rand.Reader, max)
    s.endpoint = s.Endpoints[i.Int64()]
    err := s.socket.Connect(s.endpoint)
    if err != nil {
      log.Printf("%s: Error connecting: %s\n", s.endpoint, err)
      time.Sleep(500 * time.Millisecond)
      continue
    }

    // No error, we're connected.
    s.connected = true
  }
}

func (s *FFS) fail_socket() {
  if !s.connected { return }
  s.Close()
}

func Publish(input chan []*FileEvent, server_list []string,
             server_timeout time.Duration) {
  var buffer bytes.Buffer
  //key := "abcdefghijklmnop"
  //cipher, err := aes.NewCipher([]byte(key))

  socket := FFS{
    Endpoints: server_list,
    SocketType: zmq.REQ,
    RecvTimeout: server_timeout,
    SendTimeout: server_timeout,
  }
  //defer socket.Close()

  for events := range input {
    // got a bunch of events, ship them out.
    log.Printf("Spooler gave me %d events\n", len(events))

    // Serialize with msgpack
    data, err := msgpack.Marshal(events)
    // TODO(sissel): chefk error
    _ = err
    //log.Printf("msgpack serialized %d bytes\n", len(data))

    // Compress it
    // A new compressor is used for every payload of events so
    // that any individual payload can be decompressed alone.
    // TODO(sissel): Make compression level tunable
    compressor, _ := zlib.NewWriterLevel(&buffer, 3)
    buffer.Truncate(0)
    _, err = compressor.Write(data)
    err = compressor.Flush()
    compressor.Close()
    //log.Printf("compressed %d bytes\n", buffer.Len())
    // TODO(sissel): check err

    // TODO(sissel): implement security/encryption/etc

    // Send full payload over zeromq REQ/REP
    // TODO(sissel): check error
    //buffer.Write(data)

    // Loop forever trying to send.
    // This will cause reconnects/etc on failures automatically
    for {
      err = socket.Send(buffer.Bytes(), 0)
      data, err = socket.Recv(0)
      if err == nil {
        // success!
        break
      }
    }
    // TODO(sissel): Check data value of reply?

    // TODO(sissel): retry on failure or timeout
    // TODO(sissel): notify registrar of success
  } /* for each event payload */
} // Publish



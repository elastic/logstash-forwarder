package main

import (
  "math/rand"
  "bytes"
  "encoding/binary"
  "encoding/pem"
  "crypto/tls"
  "crypto/x509"
  "net"
  "io"
  "os"
  "io/ioutil"
  "log"
  "time"
  "compress/zlib"
  "strconv"
)

var hostname string
func init() {
  log.Printf("publisher init\n")
  hostname, _ = os.Hostname()
  rand.Seed(time.Now().UnixNano())
}

func Publishv1(input chan []*FileEvent,
               registrar chan []*FileEvent,
               config *NetworkConfig) {
  var buffer bytes.Buffer
  var socket *tls.Conn
  var sequence uint32
  var err error

  socket = connect(config)
  defer socket.Close()

  for events := range input {
    buffer.Truncate(0)
    compressor, _ := zlib.NewWriterLevel(&buffer, 3)

    for _, event := range events {
      sequence += 1
      writeDataFrame(event, sequence, compressor)
    }
    compressor.Flush()
    compressor.Close()

    compressed_payload := buffer.Bytes()

    // Send buffer until we're successful...
    oops := func(err error) {
      // TODO(sissel): Track how frequently we timeout and reconnect. If we're
      // timing out too frequently, there's really no point in timing out since
      // basically everything is slow or down. We'll want to ratchet up the 
      // timeout value slowly until things improve, then ratchet it down once
      // things seem healthy.
      log.Printf("Socket error, will reconnect: %s\n", err)
      time.Sleep(1 * time.Second)
      socket.Close()
      socket = connect(config)
    }

    SendPayload: for {
      // Abort if our whole request takes longer than the configured
      // network timeout.
      socket.SetDeadline(time.Now().Add(config.timeout))

      // Set the window size to the length of this payload in events.
      _, err = socket.Write([]byte("1W"))
      if err != nil { oops(err); continue }
      binary.Write(socket, binary.BigEndian, uint32(len(events)))
      if err != nil { oops(err); continue }

      // Write compressed frame
      socket.Write([]byte("1C"))
      if err != nil { oops(err); continue }
      binary.Write(socket, binary.BigEndian, uint32(len(compressed_payload)))
      if err != nil { oops(err); continue }
      _, err = socket.Write(compressed_payload)
      if err != nil { oops(err); continue }

      // read ack
      response := make([]byte, 0, 6)
      ackbytes := 0
      for ackbytes != 6 {
        n, err := socket.Read(response[len(response):cap(response)])
        if err != nil {
          log.Printf("Read error looking for ack: %s\n", err)
          socket.Close()
          socket = connect(config)
          continue SendPayload // retry sending on new connection
        } else {
          ackbytes += n
        }
      }

      // TODO(sissel): verify ack
      // Success, stop trying to send the payload.
      break
    }

    // Tell the registrar that we've successfully sent these events
    registrar <- events
  } /* for each event payload */
} // Publish

func connect(config *NetworkConfig) (socket *tls.Conn) {
  var tlsconfig tls.Config

  if len(config.SSLCertificate) > 0 && len(config.SSLKey) > 0 {
    log.Printf("Loading client ssl certificate: %s and %s\n",
               config.SSLCertificate, config.SSLKey)
    cert, err := tls.LoadX509KeyPair(config.SSLCertificate, config.SSLKey)
    if err != nil {
      log.Fatalf("Failed loading client ssl certificate: %s\n", err)
    }
    tlsconfig.Certificates = []tls.Certificate{cert}
  }

  if len(config.SSLCA) > 0 {
    log.Printf("Setting trusted CA from file: %s\n", config.SSLCA)
    tlsconfig.RootCAs = x509.NewCertPool()

    pemdata, err := ioutil.ReadFile(config.SSLCA)
    if err != nil { log.Fatalf("Failure reading CA certificate: %s\n", err) }

    block, _ := pem.Decode(pemdata)
    if block == nil {
      log.Fatalf("Failed to decode PEM data, is %s a valid cert?\n", config.SSLCA)
    }
    if block.Type != "CERTIFICATE" {
      log.Fatalf("This is not a certificate file: %s\n", config.SSLCA)
    }

    cert, err := x509.ParseCertificate(block.Bytes)
    if err != nil {
      log.Fatalf("Failed to parse a certificate: %s\n", config.SSLCA)
    }
    tlsconfig.RootCAs.AddCert(cert)
  }

  for {
    // Pick a random server from the list.
    address := config.Servers[rand.Int() % len(config.Servers)]
    log.Printf("Connecting to %s\n", address)

    tcpsocket, err := net.DialTimeout("tcp", address, config.timeout)
    if err != nil {
      log.Printf("Failure connecting to %s: %s\n", address, err)
      time.Sleep(1 * time.Second)
      continue
    }

    socket = tls.Client(tcpsocket, &tlsconfig)
    socket.SetDeadline(time.Now().Add(config.timeout))
    err = socket.Handshake()
    if err != nil {
      log.Printf("Failed to tls handshake with %s %s\n", address, err)
      time.Sleep(1 * time.Second)
      socket.Close()
      continue
    }

    log.Printf("Connected to %s\n", address)

    // connected, let's rock and roll.
    return
  }
  return
}

func writeDataFrame(event *FileEvent, sequence uint32, output io.Writer) {
  //log.Printf("event: %s\n", *event.Text)
  // header, "1D"
  output.Write([]byte("1D"))
  // sequence number
  binary.Write(output, binary.BigEndian, uint32(sequence))
  // 'pair' count
  binary.Write(output, binary.BigEndian, uint32(len(*event.Fields) + 4))

  writeKV("file", *event.Source, output)
  writeKV("host", hostname, output)
  writeKV("offset", strconv.FormatInt(event.Offset, 10), output)
  writeKV("line", *event.Text, output)
  for k, v := range(*event.Fields) {
    writeKV(k, v, output)
  }
}

func writeKV(key string, value string, output io.Writer) {
  //log.Printf("kv: %d/%s %d/%s\n", len(key), key, len(value), value)
  binary.Write(output, binary.BigEndian, uint32(len(key)))
  output.Write([]byte(key))
  binary.Write(output, binary.BigEndian, uint32(len(value)))
  output.Write([]byte(value))
}

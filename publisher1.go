package main

import (
  "bytes"
  "compress/zlib"
  "crypto/tls"
  "crypto/x509"
  "encoding/binary"
  "encoding/pem"
  "errors"
  "fmt"
  "io"
  "io/ioutil"
  "log"
  "math/rand"
  "net"
  "os"
  "regexp"
  "strconv"
  "time"
)

var hostname string
var hostport_re, _ = regexp.Compile("^(.+):([0-9]+)$")

func init() {
  log.Printf("publisher init\n")
  hostname, _ = os.Hostname()
  rand.Seed(time.Now().UnixNano())
}

func Publishv1(input chan []*FileEvent,
  registrar chan []*FileEvent,
  config *NetworkConfig) {
  var buffer bytes.Buffer
  var compressed_payload []byte
  var socket *tls.Conn
  var last_ack_sequence uint32
  var sequence uint32
  var err error

  socket = connect(config)
  defer socket.Close()

  // TODO(driskell): Make the idle timeout configurable like the network timeout is?
  timer := time.NewTimer(900 * time.Second)

  for {
    select {
    case events := <-input:
      for {
        // Do we need to populate the buffer again? Or do we already have it done?
        if buffer.Len() == 0 {
          sequence = last_ack_sequence
          compressor, _ := zlib.NewWriterLevel(&buffer, 3)

          for _, event := range events {
            sequence += 1
            writeDataFrame(event, sequence, compressor)
          }
          compressor.Flush()
          compressor.Close()

          compressed_payload = buffer.Bytes()
        }

        // Abort if our whole request takes longer than the configured network timeout.
        socket.SetDeadline(time.Now().Add(config.timeout))

        // Set the window size to the length of this payload in events.
        _, err = socket.Write([]byte("1W"))
        if err != nil {
          log.Printf("Socket error, will reconnect: %s\n", err)
          goto RetryPayload
        }
        err = binary.Write(socket, binary.BigEndian, uint32(len(events)))
        if err != nil {
          log.Printf("Socket error, will reconnect: %s\n", err)
          goto RetryPayload
        }

        // Write compressed frame
        _, err = socket.Write([]byte("1C"))
        if err != nil {
          log.Printf("Socket error, will reconnect: %s\n", err)
          goto RetryPayload
        }
        err = binary.Write(socket, binary.BigEndian, uint32(len(compressed_payload)))
        if err != nil {
          log.Printf("Socket error, will reconnect: %s\n", err)
          goto RetryPayload
        }
        _, err = socket.Write(compressed_payload)
        if err != nil {
          log.Printf("Socket error, will reconnect: %s\n", err)
          goto RetryPayload
        }

        // read ack
        for {
          var frame [2]byte

          // Each time we've received a frame, reset the deadline
          socket.SetDeadline(time.Now().Add(config.timeout))

          err = binary.Read(socket, binary.BigEndian, &frame)
          if err != nil {
            log.Printf("Socket error, will reconnect: %s\n", err)
            goto RetryPayload
          }

          if frame == [2]byte{'1', 'A'} {
            var ack_sequence uint32

            // Read the sequence number acked
            err = binary.Read(socket, binary.BigEndian, &ack_sequence)
            if err != nil {
              log.Printf("Socket error, will reconnect: %s\n", err)
              goto RetryPayload
            }

            if sequence == ack_sequence {
              last_ack_sequence = ack_sequence
              // All acknowledged! Stop reading acks
              break
            }

            // NOTE(driskell): If the server is busy and not yet processed anything, we MAY
            // end up receiving an ack for the last sequence in the previous payload, or 0
            if ack_sequence == last_ack_sequence {
              // Just keep waiting
              continue
            } else if ack_sequence - last_ack_sequence > uint32(len(events)) {
              // This is wrong - we've already had an ack for these
              log.Printf("Socket error, will reconnect: Repeated ACK\n")
              goto RetryPayload
            }

            // Send a slice of the acknowledged events downstream and slice what we're still waiting for
            // so that if we encounter an error, we only resend unacknowledged events
            registrar <- events[:ack_sequence - last_ack_sequence]
            events = events[ack_sequence - last_ack_sequence:]
            last_ack_sequence = ack_sequence

            // Reset the events buffer so it gets regenerated if we need to retry the payload
            buffer.Truncate(0)
            continue
          }

          // Unknown frame!
          log.Printf("Socket error, will reconnect: Unknown frame received: %s\n", frame)
          goto RetryPayload
        }

        // Success, stop trying to send the payload.
        break

      RetryPayload:
        // TODO(sissel): Track how frequently we timeout and reconnect. If we're
        // timing out too frequently, there's really no point in timing out since
        // basically everything is slow or down. We'll want to ratchet up the
        // timeout value slowly until things improve, then ratchet it down once
        // things seem healthy.
        time.Sleep(1 * time.Second)
        socket.Close()
        socket = connect(config)
      }

      // Tell the registrar that we've successfully sent the remainder of the events
      registrar <- events

      // Reset the events buffer
      buffer.Truncate(0)

      // Prepare to enter idle by setting a long deadline... if we have more events we'll drop it down again
      socket.SetDeadline(time.Now().Add(1800 * time.Second))

      // Reset the timer
      timer.Reset(900 * time.Second)
    case <-timer.C:
      // We've no events to send - throw a ping (well... window frame) so our connection doesn't idle and die
      err = ping(socket)
      if err != nil {
        log.Printf("Socket error during ping, will reconnect: %s\n", err)
        time.Sleep(1 * time.Second)
        socket.Close()
        socket = connect(config)
      }

      // Reset the deadline
      socket.SetDeadline(time.Now().Add(1800 * time.Second))

      // Reset the timer
      timer.Reset(900 * time.Second)
    } /* select */
  } /* for */
} // Publish

func ping(socket *tls.Conn) error {
  var frame [2]byte

  // Set deadline for this write
  socket.SetDeadline(time.Now().Add(config.timeout))

  // This just keeps connection open through firewalls
  // We don't await for a response so its not a real ping, the protocol does not provide for a real ping
  // And with a complete replacement of protocol happening soon, makes no sense to add new frames and such
  _, err := socket.Write([]byte("1W"))
  if err != nil {
    return err
  }
  err = binary.Write(socket, binary.BigEndian, uint32(*spool-size))
  if err != nil {
    return err
  }

  return nil
}

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
    if err != nil {
      log.Fatalf("Failure reading CA certificate: %s\n", err)
    }

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
    hostport := config.Servers[rand.Int()%len(config.Servers)]
    submatch := hostport_re.FindSubmatch([]byte(hostport))
    if submatch == nil {
      log.Fatalf("Invalid host:port given: %s", hostport)
    }
    host := string(submatch[1])
    port := string(submatch[2])
    addresses, err := net.LookupHost(host)

    if err != nil {
      log.Printf("DNS lookup failure \"%s\": %s\n", host, err)
      time.Sleep(1 * time.Second)
      continue
    }

    address := addresses[rand.Int()%len(addresses)]
    addressport := fmt.Sprintf("%s:%s", address, port)

    log.Printf("Connecting to %s (%s) \n", addressport, host)

    tcpsocket, err := net.DialTimeout("tcp", addressport, config.timeout)
    if err != nil {
      log.Printf("Failure connecting to %s: %s\n", address, err)
      time.Sleep(1 * time.Second)
      continue
    }

    socket = tls.Client(tcpsocket, &tlsconfig)
    socket.SetDeadline(time.Now().Add(config.timeout))
    err = socket.Handshake()
    if err != nil {
      log.Printf("Handshake failure with %s: Failed to TLS handshake: %s\n", address, err)
      goto TryNextServer
    }

    log.Printf("Connected with %s\n", address)

    // connected, let's rock and roll.
    return

  TryNextServer:
    time.Sleep(1 * time.Second)
    socket.Close()
    continue
  }
  return
}

func writeDataFrame(event *FileEvent, sequence uint32, output io.Writer) {
  //log.Printf("event: %s\n", *event.Text)
  // header, "2D"
  // Why version 2 data frame? Because server.rb will correctly start returning partial ACKs if we specify version 2
  // This keeps the old logstash forwarders, which broke on partial ACK, working with even the newer server.rb
  // If the newer server.rb receives a 1D it will refuse to send partial ACK, just like before
  output.Write([]byte("2D"))
  // sequence number
  binary.Write(output, binary.BigEndian, uint32(sequence))
  // 'pair' count
  binary.Write(output, binary.BigEndian, uint32(len(*event.Fields)+4))

  writeKV("file", *event.Source, output)
  writeKV("host", hostname, output)
  writeKV("offset", strconv.FormatInt(event.Offset, 10), output)
  writeKV("line", *event.Text, output)
  for k, v := range *event.Fields {
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

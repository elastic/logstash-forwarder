package main

import (
  "bytes"
  "encoding/binary"
  //"crypto/tls"
  "log"
  //"time"
  "compress/zlib"
)

func init() {
}

func connect(config *NetworkConfig) {

}

func Publishv1(input chan []*FileEvent,
               registrar chan []*FileEvent,
               config *NetworkConfig) {
  var zbuf, packbuf bytes.Buffer
  socket := connect(config)
  for events := range input {

    for event := range events {
    }

    // Compress it
    // A new zlib writer  is used for every payload of events so that any
    // individual payload can be decompressed alone.
    // TODO(sissel): Make compression level tunable
    compressor, _ := zlib.NewWriterLevel(&buffer, 3)
    buffer.Truncate(0)
    _, err := compressor.Write(data)
    err = compressor.Flush()
    compressor.Close()

    // Loop forever trying to send.
    // This will cause reconnects/etc on failures automatically
    for {
      err = socket.Send(nonce, zmq.SNDMORE)
      if err != nil {
        continue // send failed, retry!
      }
      err = socket.Send(ciphertext, 0)
      if err != nil {
        continue // send failed, retry!
      }

      data, err = socket.Recv(0)
      // TODO(sissel): Figure out acknowledgement protocol? If any?
      if err == nil {
        break // success!
      }
    }

    // Tell the registrar that we've successfully sent these events
    registrar <- events
  } /* for each event payload */
} // Publish

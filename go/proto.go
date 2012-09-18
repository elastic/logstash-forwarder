package main

import (
  "crypto/tls"
  "encoding/binary"
  "fmt"
)

type Conn struct {
  tls *tls.Conn
}

type DialError struct {
  message string
  address string
  network string
}

func(err DialError) Error() (string) {
  return err.message
}

func Dial(network string, address string, config *tls.Config) (*Conn, error) {
  var err error
  conn := new(Conn)
  switch network {
    case "tls":
      conn.tls, err = tls.Dial("tcp", address, config)
    //case "tcp":
    default:
      return nil, DialError{"invalid network", address, network}
  }

  if err != nil {
    fmt.Printf("Failed to connect to %s://%s - %s\n", network, address, err)
    return nil, err
  }

  return conn, nil
} /* Dial */

func (conn *Conn) WriteFileEvent(event FileEvent) (error) {
  // V1 Data Frame
  conn.tls.Write([]byte("1D"))

  // How many fields in this data frame
  binary.Write(conn.tls, binary.BigEndian, uint32(2))

  /* write the 'path' field */
  binary.Write(conn.tls, binary.BigEndian, uint32(4))
  conn.tls.Write([]byte("path"))
  conn.tls.Write([]byte(event.path))

  /* write the 'line' field */
  binary.Write(conn.tls, binary.BigEndian, uint32(4))
  conn.tls.Write([]byte("line"))
  binary.Write(conn.tls, binary.BigEndian, uint32(len(event.line)))
  conn.tls.Write(event.line)

  return nil
} /* WriteFileEvent */

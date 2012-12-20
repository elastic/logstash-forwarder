package main
import (
  "crypto/tls"
  "net"
)

type Lumberjack struct {
  Addresses []string
  CAPath string

  sequence uint32
  conn net.Conn
  tls *tls.Conn
  tlsconf *tls.Config
}


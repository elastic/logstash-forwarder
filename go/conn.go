package main

import (
  "fmt"
  "net"
  "crypto/tls"
)

func (l *Lumberjack) connected() bool {
  return l.conn != nil
}

func (l *Lumberjack) connect_once() (err error) {
  if l.connected() { l.disconnect() }

  if l.tlsconf == nil {
    l.tlsconf, err = configureTLS(l.CAPath)
    if err != nil {
      return
    }
  }

  l.conn, err = net.Dial("tcp", l.Addresses[0])
  if err != nil { return }

  l.tls = tls.Client(l.conn, l.tlsconf)
  err = l.tls.Handshake()
  if err != nil { return }
  return
}

/* Connect to a remote lumberjack server. This blocks until the connection is
 * ready. It will retry until successful. */
func (l *Lumberjack) connect() {
  for !l.connected() {
    err := l.connect_once()
    if err != nil {
      fmt.Printf("Error connecting: %s\n", err)
    }
  }
} /* Lumberjack#connect */


/* Disconnect from the lumberjack server */
func (l *Lumberjack) disconnect() {
  if !l.connected() { return }
  l.tls.Close()
  l.conn.Close()
  l.tls = nil
  l.conn = nil
} /* Lumberjack#disconnect */

func (l *Lumberjack) publish(event map[string]string) {
  if !l.connected() { l.connect() }

  writeData(l.tls, 1, event)
}

func main() {
  l := Lumberjack{Addresses: []string{"localhost:4000"}, CAPath: "/home/jls/projects/logstash/server.crt"}

  l.publish(map[string]string{ "hello": "world"})
  l.disconnect()
}

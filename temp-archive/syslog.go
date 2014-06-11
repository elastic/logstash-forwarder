// +build !windows

package main

import (
  "log"
  "log/syslog"
)

func configureSyslog() {
  writer, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "logstash-forwarder")
  if err != nil {
    log.Fatalf("Failed to open syslog: %s\n", err)
    return
  }
  log.SetOutput(writer)
}

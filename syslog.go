// +build !windows

package main

import (
	"log"
	"log/syslog"
)

func configureSyslog() (*syslog.Writer, error) {
	writer, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "logstash-forwarder")
	if err != nil {
		log.Fatalf("Failed to open syslog: %s\n", err)
	}
	return writer, nil
}

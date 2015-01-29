#!/bin/sh

if [ "$1" = configure ] ; then
  SHOULD_START='no'

  # If we upgrade from lumberjack to logstash-forwarder, the
  # pre-install script created this file if lumberjack was
  # running. So start logstash-forwarder.
  if [ -f /tmp/lumberjack-running ]; then
    rm -f /tmp/lumberjack-running
    SHOULD_START='yes'
  fi

  # If logstash-forwarder is running, than restart it afterward
  /usr/sbin/service logstash-forwarder status > /dev/null 2>&1
  if [ $? -eq 0 ]; then
    SHOULD_START='yes'
  fi

  if [ $SHOULD_START = 'yes' ]; then
    /usr/sbin/service logstash-forwarder restart
  fi

  exit 0
fi

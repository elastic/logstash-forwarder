#!/bin/sh

if [ "$1" = "remove" ] ; then
  /usr/sbin/service logstash-forwarder stop
  sleep 2
  # use a jack-hammer
  [ -x '/usr/bin/pkill' ] && /usr/bin/pkill -9 logstash-forwarder >/dev/null 2>&1
  exit 0
fi

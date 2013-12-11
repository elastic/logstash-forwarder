#!/bin/sh

set -e
export PATH='/bin:/sbin:/usr/bin:/usr/sbin'

stop_service() {
  service $1 stop
  sleep 2
  # use a jack-hammer
  [ -x '/usr/bin/pkill' ] && /usr/bin/pkill -9 $1 >/dev/null 2>&1
  exit 0
}

if [ "$1" = remove ] ; then
  for x in {lumberjack,logstash-forwarder}; do
    check_service $x
  done
fi

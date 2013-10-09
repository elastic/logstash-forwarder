#!/bin/sh

set -e
export PATH='/bin:/sbin:/usr/bin:/usr/sbin'

if [ "$1" = remove ] ; then
  service lumberjack stop
  sleep 2
  # use a jack-hammer
  [ -x '/usr/bin/pkill' ] && /usr/bin/pkill -9 lumberjack >/dev/null 2>&1
  exit 0
fi

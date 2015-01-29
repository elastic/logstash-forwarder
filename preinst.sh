#!/bin/sh

if [ "$1" = "upgrade"]; then
  # If we upgrade from lumberjack to logstash-forwarder,
  # we still want to know if we need to start the service
  # after the init-script is replaced.
  if [ -f /etc/init.d/lumberjack ]; then
    # The 'old' init file exists now, stop the service
    /usr/sbin/service lumberjack status >/dev/null 2>&1
    if [ $? -eq 0 ]; then
      /bin/touch /tmp/lumberjack-running
      /usr/sbin/service lumberjack stop
      sleep 2
      # use a jack-hammer
      [ -x '/usr/bin/pkill' ] && /usr/bin/pkill -9 $1 >/dev/null 2>&1
      exit 0
    fi
  fi
fi

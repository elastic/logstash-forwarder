#!/bin/sh

if [ $1 = "remove" ]; then
  service logstash-forwarder stop >/dev/null 2>&1 || true
  update-rc.d -f logstash-forwarder remove

  if getent passwd logstash-forwarder >/dev/null ; then
    userdel logstash-forwarder
  fi

  if getent group logstash-forwarder >/dev/null ; then
    groupdel logstash-forwarder
  fi
fi

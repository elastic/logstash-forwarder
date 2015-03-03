#!/bin/sh

# create logstash-forwarder group
if ! getent group logstash-forwarder >/dev/null; then
  groupadd -r logstash-forwarder
fi

# create logstash-forwarder user
if ! getent passwd logstash-forwarder >/dev/null; then
  useradd -M -r -g logstash-forwarder -d /var/lib/logstash-forwarder \
    -s /usr/sbin/nologin -c "logstash-forwarder Service User" logstash-forwarder
fi

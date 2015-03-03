# create logstash-forwarder group
if ! getent group logstash-forwarder >/dev/null; then
  groupadd -r logstash-forwarder
fi

# create logstash-forwarder user
if ! getent passwd logstash-forwarder >/dev/null; then
  useradd -r -g logstash-forwarder -d /opt/logstash-forwarder \
    -s /sbin/nologin -c "logstash-forwarder" logstash-forwarder
fi

#!/bin/sh

chown -R logstash-forwarder:logstash-forwarder /opt/logstash-forwarder
chown logstash-forwarder /var/log/logstash-forwarder
chown logstash-forwarder:logstash-forwarder /var/lib/logstash-forwarder
update-rc.d logstash-forwarder defaults
if [ -f /etc/logstash-forwarder.conf ]; then
  echo "Found /etc/logstash-forwarder.conf.  Moving to /etc/logstash-forwarder.conf.d/logstash-forwarder.conf"
  mv /etc/logstash-forwarder.conf /etc/logstash-forwarder.conf.d
  echo "restarting logstash-forwarder with the new config"
  service logstash-forwarder restart
fi
echo "Logs for logstash-forwarder will be in /var/log/logstash-forwarder/"

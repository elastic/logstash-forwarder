/sbin/chkconfig --add logstash-forwarder

chown -R logstash-forwarder:logstash-forwarder /opt/logstash-forwarder
chown logstash-forwarder /var/log/logstash-forwarder
chown logstash-forwarder:logstash-forwarder /var/lib/logstash-forwarder

echo "Logs for logstash-forwarder will be in /var/log/logstash-forwarder/"

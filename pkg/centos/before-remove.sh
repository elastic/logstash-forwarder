if [ $1 -eq 0 ]; then
  /sbin/service logstash-forwarder stop >/dev/null 2>&1 || true
  /sbin/chkconfig --del logstash-forwarder
  if getent passwd logstash-forwarder >/dev/null ; then
    userdel logstash-forwarder
  fi

  if getent group logstash-forwarder > /dev/null ; then
    groupdel logstash-forwarder
  fi
fi

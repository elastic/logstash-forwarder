#!/bin/sh

export PATH='/bin:/sbin:/usr/bin:/usr/sbin'

check_service () {
  # In-between solution whilst renaming
  service $1 status > /dev/null 2>&1
  case $? in
    0) # Running
       service $1 restart
       exit $?
       ;;
    *) exit 0;; # not running so just don't do anything
  esac
}

if [ "$1" = configure ] ; then
  for x in {lumberjack,logstash-forwarder}; do
    check_service $x
  done
fi

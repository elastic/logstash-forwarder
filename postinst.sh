#!/bin/sh

export PATH='/bin:/sbin:/usr/bin:/usr/sbin'

if [ "$1" = configure ] ; then
  service lumberjack status > /dev/null 2>&1
  case $? in
    0) # Running
       service lumberjack restart
       exit $?
       ;;
    *) exit 0;; # not running so just don't do anything
  esac
fi

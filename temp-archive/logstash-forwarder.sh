#!/bin/sh

dir=`dirname $0`
LD_LIBRARY_PATH="${dir}/../lib"
export LD_LIBRARY_PATH
exec "${dir}/logstash-forwarder" "$@"

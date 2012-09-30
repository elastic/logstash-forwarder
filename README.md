# lumberjack

o/~ I'm a lumberjack and I'm ok! I sleep when idle, then I ship logs all day! I parse your logs, I eat the JVM agent for lunch! o/~

Collect logs locally in preparation for processing elsewhere!

Problem: logstash jar releases are too fat for constrained systems.

## Goals

* minimize resource usage where possible (cpu, memory, network)
* secure transmission of logs
* configurable event data
* easy to deploy with minimal moving parts.

Simple inputs only:

* follow files, respect rename/truncation conditions
* stdin, useful for things like 'varnishlog | lumberjack ...'

## Implementation details 

Below is valid as of 2012/09/19

### Minimize resource usage

* sets small resource limits (memory, open files) on start up based on the number of files being watched
* cpu: sleeps when there is nothing to do
* network/cpu: sleeps if there is a network failure
* network: uses zlib for compression

### secure transmission

* uses openssl to transport logs. Currently supports verifying the server
  certificate only (so you know who you are sending to).

### configurable event data

* the protocol lumberjack uses supports sending a string:string map
* the lumberjack tool lets you specify arbitrary extra data with `--field name=value`

## easy deployment

* all dependencies are built at compile-time (openssl, jemalloc, etc)
* 'make deb' (or make rpm) will package everything into a single deb (or rpm)
* bin/lumberjack.sh makes sure the dependencies are found

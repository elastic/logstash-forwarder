# lumberjack

o/~ I'm a lumberjack and I'm ok! I sleep when idle, then I ship logs all day! I parse your logs, I eat the JVM agent for lunch! o/~

## QUESTIONS?

If you have questions and cannot find answers, please join the #logstash irc
channel on freenode irc or ask on the logstash-users@googlegroups.com mailing
list.

## What is this?

A tool to collect logs locally in preparation for processing elsewhere!

Problem: logstash jar releases are too fat for constrained systems.

Solution: lumberjack

## Building it

* compile: make 
* rpm package: make rpm
* deb package: make deb

Packages install to /opt/lumberjack. Lumberjack builds all necessary
dependencies itself, so there should be no run-time dependencies you
need.

## Running it

Generally: `lumberjack.sh --host somehost --port 12345 /var/log/messages`

You'll need an ssl ca to verify the server (host) with.

See `lumberjack.sh --help`

## Use with logstash

In logstash, you'll want to use the [lumberjack](http://logstash.net/docs/latest/inputs/lumberjack) input, something like:

    input {
      lumberjack {
        # The port to listen on
        port => 12345

        # The paths to your ssl cert and key
        ssl_certificate => "path/to/ssl.crt"
        ssl_key => "path/to/ssl.key"

        # Set this to whatever you want.
        type => "somelogs"
      }
    }

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

* sets small resource limits (memory, open files) on start up based on the
  number of files being watched
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

* all dependencies are built at compile-time (openssl, jemalloc, etc) because many os distributions lack these dependencies.
* 'make deb' (or make rpm) will package everything into a single deb (or rpm)
* bin/lumberjack.sh makes sure the dependencies are found when run in production

## future functional features

* re-evaluate globs periodically to look for new log files
* track position of in the log

## future protocol discussion

I would love to not have a custom protocol, but nothing I've found implements
what I need, which is: encrypted, trusted, compressed, latency-resilient, and
reliable transport of events.

* redis development refuses to accept encryption support, would likely reject
  compression as well.
* zeromq lacks authentication, encryption, and compression.
* thrift also lacks authentication, encryption, and compression, and also is an
  RPC framework, not a streaming system.
* websockets don't do authentication or compression, but support encrypted
  channels with SSL. Websockets also require XORing the entire payload of all
  messages - wasted energy.
* SPDY is still changing too frequently and is also RPC. Streaming requires
  custom framing.
* HTTP is RPC and very high over head for small events (uncompressable headers,
  etc). Streaming requires custom framing.

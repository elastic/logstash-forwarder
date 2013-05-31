# lumberjack

o/~ I'm a lumberjack and I'm ok! I sleep when idle, then I ship logs all day! I parse your logs, I eat the JVM agent for lunch! o/~

## Questions and support

If you have questions and cannot find answers, please join the #logstash irc
channel on freenode irc or ask on the logstash-users@googlegroups.com mailing
list.

## What is this?

A tool to collect logs locally in preparation for processing elsewhere!

Problem: logstash jar releases are too fat for constrained systems.

Solution: lumberjack

### Goals

* Minimize resource usage where possible (CPU, memory, network).
* Secure transmission of logs.
* Configurable event data.
* Easy to deploy with minimal moving parts.
* Simple inputs only:
  * Follows files and respects rename/truncation conditions.
  * Accepts `STDIN`, useful for things like `varnishlog | lumberjack...`.

## Building it

1. Install [FPM](https://github.com/jordansissel/fpm)

        sudo gem install fpm

2. Ensure you have outging FTP access to download OpenSSL from
`ftp.openssl.org`.

3. Compile lumberjack

        git clone git://github.com/jordansissel/lumberjack.git
        cd lumberjack
        make

4. Make packages, either:

        make rpm

    Or:

        make deb

## Installing it

Packages install to `/opt/lumberjack`. Lumberjack builds all necessary
dependencies itself, so there should be no run-time dependencies you
need.

## Running it

Generally:

    lumberjack.sh --host somehost --port 12345 /var/log/messages

See `lumberjack.sh --help` for all the flags

### Key points

* You'll need an SSL CA to verify the server (host) with.
* You can specify custom fields with the `--field foo=bar`. Any number of these
  may be specified. I use them to set fields like `type` and other custom
  attributes relevant to each log.
* Any non-flag argument after is considered a file path. You can watch any
  number of files.

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

## Implementation details 

Below is valid as of 2012/09/19

### Minimize resource usage

* Sets small resource limits (memory, open files) on start up based on the
  number of files being watched.
* CPU: sleeps when there is nothing to do.
* Network/CPU: sleeps if there is a network failure.
* Network: uses zlib for compression.

### Secure transmission

* Uses OpenSSL to verify the server certificates (so you know who you
  are sending to).
* Uses OpenSSL to transport logs.

### Configurable event data

* The protocol lumberjack uses supports sending a `string:string` map.
* The lumberjack tool lets you specify arbitrary extra data with
  `--field name=value`.

### Easy deployment

* All dependencies are built at compile-time (OpenSSL, jemalloc, etc) because many os distributions lack these dependencies.
* The `make deb` or `make rpm` commands will package everything into a
  single DEB or RPM.
* The `bin/lumberjack.sh` script makes sure the dependencies are found
  when run in production.

### Future functional features

* Re-evaluate globs periodically to look for new log files.
* Track position of in the log.

### Future protocol discussion

I would love to not have a custom protocol, but nothing I've found implements
what I need, which is: encrypted, trusted, compressed, latency-resilient, and
reliable transport of events.

* Redis development refuses to accept encryption support, would likely reject
  compression as well.
* ZeroMQ lacks authentication, encryption, and compression.
* Thrift also lacks authentication, encryption, and compression, and also is an
  RPC framework, not a streaming system.
* Websockets don't do authentication or compression, but support encrypted
  channels with SSL. Websockets also require XORing the entire payload of all
  messages - wasted energy.
* SPDY is still changing too frequently and is also RPC. Streaming requires
  custom framing.
* HTTP is RPC and very high overhead for small events (uncompressable headers,
  etc). Streaming requires custom framing.

## License 

See LICENSE file.


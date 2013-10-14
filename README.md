# lumberjack

o/~ I'm a lumberjack and I'm ok! I sleep when idle, then I ship logs all day! I parse your logs, I eat the JVM agent for lunch! o/~

## Questions and support

If you have questions and cannot find answers, please join the #logstash irc
channel on freenode irc or ask on the logstash-users@googlegroups.com mailing
list.

## What is this?

A tool to collect logs locally in preparation for processing elsewhere!

Problem: logstash jar releases are too fat for constrained systems. Until we can comfortably promise logstash executing with less resource usage...

Solution: lumberjack

## Configuring

lumberjack is configured with a json file you specify with the -config flag:

`lumberjack -config yourstuff.json`

Here's a sample, with comments in-line to describe the settings. Please please
please keep in mind that comments are technically invalid in JSON, so you can't
include them in your config.:

    {
      # The network section covers network configuration :)
      "network": {
        # A list of downstream servers listening for our messages.
        # lumberjack will pick one at random and only switch if
        # the selected one appears to be dead or unresponsive
        "servers": [ "localhost:5043" ],

        # The path to your client ssl certificate (optional)
        "ssl certificate": "./lumberjack.crt",
        # The path to your client ssl key (optional)
        "ssl key": "./lumberjack.key",

        # The path to your trusted ssl CA file. This is used
        # to authenticate your downstream server.
        "ssl ca": "./lumberjack_ca.crt",

        # Network timeout in seconds. This is most important for lumberjack
        # determining whether to stop waiting for an acknowledgement from the
        # downstream server. If an timeout is reached, lumberjack will assume
        # the connection or server is bad and will connect to a server chosen
        # at random from the servers list.
        "timeout": 15
      },

      # The list of files configurations
      "files": [
        # An array of hashes. Each hash tells what paths to watch and
        # what fields to annotate on events from those paths.
        {
          "paths": [ 
            # single paths are fine
            "/var/log/messages",
            # globs are fine too, they will be periodically evaluated
            # to see if any new files match the wildcard.
            "/var/log/*.log"
          ],

          # A dictionary of fields to annotate on each event.
          "fields": { "type": "syslog" }
        }, {
          # A path of "-" means stdin.
          "paths": [ "-" ],
          "fields": { "type": "stdin" }
        }, {
          "paths": [
            "/var/log/apache/httpd-*.log"
          ],
          "fields": { "type": "apache" }
        }
      ]
    }

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

        $ sudo gem install fpm

2. Install [go](http://golang.org/doc/install)


3. Compile lumberjack

        $ git clone git://github.com/jordansissel/lumberjack.git
        $ cd lumberback
        $ make

4. Make packages, either:

        $ make rpm

    Or:

        $ make deb

## Installing it

Packages install to `/opt/lumberjack`. Lumberjack builds all necessary
dependencies itself, so there should be no run-time dependencies you
need.

## Running it

Generally:

    $ lumberjack.sh -config lumberjack.conf

See `lumberjack.sh -help` for all the flags

The config file is documented further up in this file.

### Key points

* You'll need an SSL CA to verify the server (host) with.
* You can specify custom fields for each set of paths in the config file. Any
  number of these may be specified. I use them to set fields like `type` and
  other custom attributes relevant to each log.

### Generating an ssl certificate

Logstash supports all certificates, including self-signed certificates. To generate a certificate, you can run the following command:

    $ openssl req -x509 -batch -nodes -newkey rsa:2048 -keyout lumberjack.key -out lumberjack.crt

This will generate a key at `lumberjack.key` and the certificate at `lumberjack.crt`. Both the server that is running lumberjack as well as the logstash instances receiving logs will require these files on disk to verify the authenticity of messages.

Recommended file locations:

- certificates: `/etc/pki/tls/certs`
- keys: `/etc/pki/tls/private`

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


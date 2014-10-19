# logstash-forwarder

♫ I'm a lumberjack and I'm ok! I sleep when idle, then I ship logs all day! I parse your logs, I eat the JVM agent for lunch! ♫

(This project was recently renamed from 'lumberjack' to 'logstash-forwarder' to
make its intended use clear. The 'lumberjack' name now remains as the network protocol, and 'logstash-forwarder' is the name of the program. It's still the same lovely log forwarding program you love.)

## Questions and support

If you have questions and cannot find answers, please join the #logstash irc
channel on freenode irc or ask on the logstash-users@googlegroups.com mailing
list.

## What is this?

A tool to collect logs locally in preparation for processing elsewhere!

### Resource Usage Concerns

Perceived Problems: Some users view logstash releases as "large" or have a generalized fear of Java.

Actual Problems: Logstash, for right now, runs with a footprint that is not
friendly to underprovisioned systems such as EC2 micro instances; on other
systems it is fine. This project will exist until that is resolved.

### Transport Problems

Few log transport mechanisms provide security, low latency, and reliability.

The lumberjack protocol used by this project exists to provide a network
protocol for transmission that is secure, low latency, low resource usage, and
reliable.

## Configuring

logstash-forwarder is configured with a json file you specify with the -config flag:

`logstash-forwarder -config yourstuff.json`

Here's a sample, with comments in-line to describe the settings. Comments are
invalid in JSON, but logstash-forwarder will strip them out for you if they're
the only thing on the line:

    {
      # The network section covers network configuration :)
      "network": {
        # A list of downstream servers listening for our messages.
        # logstash-forwarder will pick one at random and only switch if
        # the selected one appears to be dead or unresponsive
        "servers": [ "localhost:5043" ],

        # The path to your client ssl certificate (optional)
        "ssl certificate": "./logstash-forwarder.crt",
        # The path to your client ssl key (optional)
        "ssl key": "./logstash-forwarder.key",

        # The path to your trusted ssl CA file. This is used
        # to authenticate your downstream server.
        "ssl ca": "./logstash-forwarder.crt",

        # Network timeout in seconds. This is most important for
        # logstash-forwarder determining whether to stop waiting for an
        # acknowledgement from the downstream server. If an timeout is reached,
        # logstash-forwarder will assume the connection or server is bad and
        # will connect to a server chosen at random from the servers list.
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

You can also read an entire directory of JSON configs by specifying a directory instead of a file with the `-config` option.

### Goals

* Minimize resource usage where possible (CPU, memory, network).
* Secure transmission of logs.
* Configurable event data.
* Easy to deploy with minimal moving parts.
* Simple inputs only:
  * Follows files and respects rename/truncation conditions.
  * Accepts `STDIN`, useful for things like `varnishlog | logstash-forwarder...`.

## Building it

1. Install [go](http://golang.org/doc/install)

2. Compile logstash-forwarder

        git clone git://github.com/elasticsearch/logstash-forwarder.git
        cd logstash-forwarder
        go build

## Packaging it (optional)

You can make native packages of logstash-forwarder.

To build the packages, you will need ruby and fpm installed.

    gem install fpm

Now build an rpm:

        make rpm

Or:

        make deb

## Installing it (via packages only)

If you don't use rpm or deb make targets as above, you can skip this section.

Packages install to `/opt/logstash-forwarder`.

There are no run-time dependencies.

## Running it

Generally:

    logstash-forwarder -config logstash-forwarder.conf

See `logstash-forwarder -help` for all the flags. The `-config` option is required and logstash-forwrder will not run without it.

The config file is documented further up in this file.

And also note that logstash-forwarder runs quietly when all is a-ok. If you want informational feedback, use the `verbose` flag to enable log emits to stdout.

Fatal errors are always sent to stderr regardless of the `-verbose` command-line option and process exits with a non-zero status.

### Key points

* You'll need an SSL CA to verify the server (host) with.
* You can specify custom fields for each set of paths in the config file. Any
  number of these may be specified. I use them to set fields like `type` and
  other custom attributes relevant to each log.

### Generating an ssl certificate

Logstash supports all certificates, including self-signed certificates. To generate a certificate, you can run the following command:

    $ openssl req -x509 -batch -nodes -newkey rsa:2048 -keyout logstash-forwarder.key -out logstash-forwarder.crt -days 365

This will generate a key at `logstash-forwarder.key` and the 1-year valid certificate at `logstash-forwarder.crt`. Both the server that is running logstash-forwarder as well as the logstash instances receiving logs will require these files on disk to verify the authenticity of messages. 

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

* The protocol supports sending a `string:string` map.

### Easy deployment

* The `make deb` or `make rpm` commands will package everything into a
  single DEB or RPM.

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


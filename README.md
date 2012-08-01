# lumberjack

Collect logs locally in preparation for processing elsewhere!

Problem: logstash jar releases are too fat for constrained systems.

Goal: Something small, fast, and light-weight to ship local logs externally.

## Requirements

* minimal resource usage
* configurable event data
* encryption and compression

Simple inputs only:

* follow files, respect rename/truncation conditions
* local sockets, maybe, if syslog(3) is worth supporting.
* stdin, useful for things like 'varnishlog | lumberjack ...'

Simple outputs only:

* custom wire event protocol (TBD)

## Tentative idea:

    # Ship apache logs in real time to somehost:12345
    ./lumberjack --target somehost:12345 /var/log/apache/access.log ...

    # Ship apache logs with additional log fields:
    ./lumberjack --target foo:12345 --field host=$HOSTNAME --field role=apt-repo /mnt/apt/access.log

* Serialization: msgpack (likely)
* Encryption: SSL
* Authentication (both directions): SSL certificates
* Compression: TLS v1 comes with compression, might be sufficient.

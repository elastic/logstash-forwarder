# lumberjack

Collect logs locally in preparation for processing elsewhere!

Problem: logstash jar releases are too fat for constrained systems.

Goal: Something small, fast, and light-weight to ship local logs externally.

## Requirements

* minimal resources
* configurable event data

Simple inputs only:

* follow files, respect rename/truncation conditions
* local sockets

Simple outputs only:

* custom wire event protocol (TBD)

## Tentative idea:

    # Ship apache logs in real time to somehost:12345
    ./lumberjack --target somehost:12345 /var/log/apache/access.log ...

    # Ship apache logs with additional log fields:
    ./lumberjack --target foo:12345 --field host=$HOSTNAME --field role=apt-repo /mnt/apt/access.log

Wire protocol will be msgpack for speed of parsing unless I find something
faster that's easy to use in as many languages.

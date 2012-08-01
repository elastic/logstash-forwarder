# lumberjack

Collect logs locally in preparation for processing elsewhere!

Problem: logstash jar releases are too fat for constrained systems.

Goal: Something small, fast, and light-weight to ship local logs externally.

## Design

* minimal resources

Simple inputs only:

* follow files, respect rename/truncation conditions
* local sockets

Simple outputs only:

* custom wire event protocol (TBD)

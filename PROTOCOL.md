# Protocol

## Goals

* simple to code, light on cpu and bandwidth reading/writing the wire protocol
* message-oriented
* lossless transmission (application-level acknowledgements, retransmit, timeout+redirect)
* protected (encryption, authentication)
* ordered
* scalable (load balance, etc)

## Prior Art in network protocols

* TCP, SCTP, WebSockets, HTTP, TLS, SSH
* WebSockets are fail because almost no load loadbalancers support HTTP Upgrade.
* Quake 3: http://fabiensanglard.net/quake3/network.php
* SCTP is fail because most folks don't understand how to firewall it and it's
  not supported on (any?) cloud stuff.
* HTTP is request/response with high overhead for bidirectional communication.
* TLS is a good framework to sit on to get encryption and authentication.
* SSH v2 channels are pretty neat. Also solves encryption + authentication.

## Questions:

* Permit bulk acknowlegements? Like TCP, perhaps.
* Should authentication be channel- or message-based?

## Tentative Plan

* Authentication: ssl certs
* Encryption: tls
* Compression, maybe? gzip (most common)

I'd rather not invent my own serialization for the protocol, but everything
else seems rather awkward to use. Protobufs are C++ (not C), msgpack may be
awkward for distribution, thrift is C++, etc.

## Implementation Considerations

### Simple/Few/Fast Dependencies

* Serialization: msgpack, json, thrift, and protobufs are all too hard to
  integrate/deploy or are too slow/complex to generate (json).
* Encryption: openssl is fairly ubiquitous and nontrivial to reimplement.
* Framing: version, frame type, payload.

### Small CPU-cost

* Serialization and Framing should be cheap on cpu. This means avoiding
  serialization mechanisms that inspect and possibly modify every single byte
  of a string (Example of expensive serialization: json's UTF-8 + escape code
  enforcement).

# Lumberjack Protocol (Still in development)

## Behavior

Sequence and ack behavior (including sliding window, etc) is similar to TCP,
but instead of bytes, messages are the base unit.

A writer with a window size of 50 events can send up to 50 unacked events
before blocking. A reader can acknowledge the 'last event' received to
support bulk acknowledgements.

Reliable, ordered byte transport is ensured by using TCP (or TLS on top), and
this protocol aims to provide reliable, application-level, message transport.

## Wire Format

### Framing

      0                   1                   2                   3
      0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
     +---------------+---------------+-------------------------------+
     |   version     |   frame type  |     payload ...               |
     +---------------------------------------------------------------+
     |   payload continued...                                        |
     +---------------------------------------------------------------+

### 'data' frame type

* SENT FROM WRITER ONLY
* frame type value: ASCII 'D' aka byte value 0x44

data is a map of string:string pairs. This is analogous to a Hash in Ruby, a
JSON map, etc, but only strings are supported at this time.

Payload:

* 32bit unsigned sequence number
* 32bit 'pair' count (how many key/value sequences follow)
* 32bit unsigned key length followed by that many bytes for the key
* 32bit unsigned value length followed by that many bytes for the value
* repeat key/value 'count' times.

* TODO(sissel): What happens when the sequence number rolls over?
* TODO(sissel): Worth supporting numerical value items instead of just strings?

### 'ack' frame type

* SENT FROM READER ONLY
* frame type value: ASCII 'A' aka byte value 0x41

Payload:

* 32bit unsigned sequence number.

Bulk acks are supported. If you receive data frames in sequence order
1,2,3,4,5,6, you can send an ack for '6' and the writer will take this to
mean you are acknowledging all data frames before and including '6'.

### 'window size' frame type

* SENT FROM WRITER ONLY
* frame type value: ASCII 'W' aka byte value 0x57

Payload:

* 32bit unsigned window size value in units of whole data frames.

This frame is used to tell the reader the maximum number of unacknowledged
data frames the writer will send before blocking for acks.

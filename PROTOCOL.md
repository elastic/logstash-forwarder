# Protocol

## Goals

* small cpu cost to encode/decode
* easy deployment (simple and/or few dependencies)
* small bandwidth cost to transmit
* message-oriented
* lossless transmission (application-level acknowledgements, retransmit, timeout+redirect)
* protected (encryption, authentication)
* ordered
* scalable (load balance, etc)

* needs to be easy to code into logstash and other projects

## Prior Art in network protocols

* http://fabiensanglard.net/quake3/network.php
* TCP, SCTP, WebSockets, HTTP, TLS, SSH
* WebSockets are fail, because almost no load loadbalancers support HTTP Upgrade.
* SCTP is fail, because most folks don't understand how to firewall it and it's
  not supported on (any?) cloud stuff.
* HTTP is request/response with high overhead.
* TLS is a good framework to sit on to get encryption and authentication.
* SSH v2 channels are pretty neat. Also solves encryption + authentication.

## Questions:

* Permit bulk acknowlegements? Like TCP, perhaps.
* Should authentication be channel- or message-based?

## Tentative Plan

* Messaging: Length-known messages sent over an encrypted TLS channel. Messages
  have a sequence id.
* Serialization: versioned, minimal+documented map-like string:string serialization.
* Authentication: ssl certs
* Encryption: tls
* Compression: gzip (most common)

I'd rather not invent my own serialization for the protocol, but everything
else seems rather awkward to use. Protobufs are C++ (not C), msgpack may be
awkward and/or slow in Java, thrift is C++, etc.

## Implementation Considerations

### Simple/Few/Fast Dependencies

* Serialization: msgpack, json, thrift, and protobufs are all too hard to
  integrate/deploy or are too slow/complex to generate (json).
* Encryption: openssl is fairly ubiquitous and nontrivial to reimplement.
* Framing: version, frame type, payload.

### Small CPU-cost

* Serialization and Framing should be cheap on cpu. This means avoiding
  serialization mechanisms that inspect and possibly modify every single byte
  of a string (json's UTF-8 + escape code enforcement).

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

data is a map of string:string pairs. This is analogous to a Hash in Ruby, a
JSON map, etc, but only strings are supported at this time.

* frame type value: ASCII 'D' aka byte value 0x44

Payload:

* 32bit unsigned sequence number
* 32bit 'pair' count (how many key/value sequences follow)
* 32bit unsigned key length followed by that many bytes for the key
* 32bit unsigned value length followed by that many bytes for the value
* repeat key/value 'count' times.

### 'ack' frame type

* frame type value: ASCII 'A' aka byte value 0x41

Payload:

* 32bit unsigned sequence number.

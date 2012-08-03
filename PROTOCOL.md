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

* needs to be easy to integrate into logstash

## Questions:

* Permit bulk acknowlegements? Like TCP, perhaps.
* Should authentication be channel- or message-based?

## Tentative Plan

* Messaging: Length-known messages sent over an encrypted TLS channel. Messages
  have a sequence id.
* Serialization: versioned, minimal+documented map-like string:string serialization.
* Authentication: ssl certs.
* Encryption: tls
* Compression: gzip (most common)

## Implementation Considerations

### Simple/Few/Fast Dependencies

* Serialization: msgpack, json, thrift, and protobufs are all too hard to
  integrate/deploy or are too slow/complex to generate (json).
* Serialization: Maybe msgpack, it's the best (simple/easy/fast) of all options
  it seems.
* Framing: zeromq can be easily vendored
* Encryption: openssl is fairly ubiquitous and nontrivial to reimplement.

### Small CPU-cost

* Serialization and Framing should be cheap on cpu. This means avoiding
  serialization mechanisms that inspect and possibly modify every single byte
  of a string (json's UTF-8 + escape code enforcement).


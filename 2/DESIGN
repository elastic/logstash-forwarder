Same basic functional design as the original lumberjack:
  Harvester reads events from files
    Event:
      Byte offset of start of event (number)
      Line number of event (number)
      File origin of event (string)
      Message (string)

  Work model:
    Harvester(s) 
      -> Enveloper (flush when full or after N idle seconds)
        -> Compressor (compresses whole envelopes)
          -> Encryptor (encrypts compressed envelopes)
            -> Emitter (ships over the wire)

Sending an envelope of an encrypted, compressed batch of messages allows
me freedom to pick any message-oriented protocol. The previous implementation
of lumberjack requried channel-encryption (with tls) which limited the 
kind of transportation tools.

Previously, compression was done on envelopes, but TLS was used to communicate
securely.

Messaging model w/ ZMQ:
  * REQREP message model
    REQREP has high latency (lock step request-response) but since
    I'm sending multiple events at once, I believe that latency is
    unimportant.

Messaging model w/ Redis:
  * RPUSH + LPOP
  * PUBLISH + SUBSCRIBE

Types of events:
  File Event - represents an event read from a file
    - file origin of event
    - byte offset of event
    - line number of event
    - event message (the contents)
  Compressed Envelope
    - number of items
    - type of item
    - compressed payload
  Encrypted Envelope
    - cipher
    - payload

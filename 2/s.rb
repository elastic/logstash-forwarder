require "ffi-rzmq"
require "zlib"
require "msgpack"

c = ZMQ::Context.new
s = c.socket(ZMQ::REP)

s.bind("tcp://*:5005")

msg = ""
msg.force_encoding("BINARY")
start = Time.now
count = 0
loop do
  rc = s.recv_string(msg)
  #p msg
  original = Zlib::Inflate.inflate(msg)
  events = MessagePack.unpack(original)
  count += events.count
  p events
  #if count > 100000
    #puts count / (Time.now - start)
    #count = 0
    #start = Time.now
  #end
  s.send_string("")
end

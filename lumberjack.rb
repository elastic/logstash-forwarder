#!/usr/bin/env ruby

require "socket"
require "thread"
require "openssl"

Thread.abort_on_exception = true

def handle(fd)
  last_ack = 0
  window_size = 64
  while true
    version = fd.sysread(1)
    #puts "version: #{version}"
    frame = fd.sysread(1)
    #puts "frame: #{frame}"

    if frame != "D"
      puts "Unexpected frame type: #{frame}"
      fd.close
      return
    end

    # data frame
    sequence = fd.sysread(4).unpack("N").first
    #puts "seq: #{sequence}"
    count = fd.sysread(4).unpack("N").first
    #puts "count: #{count}"

    map = {}
    count.times do 
      key_len = fd.sysread(4).unpack("N").first
      key = fd.sysread(key_len);
      value_len = fd.sysread(4).unpack("N").first
      value = fd.sysread(value_len);
      map[key] = value
    end
    #p sequence => map

    if sequence - last_ack >= window_size
      # ack this.
      fd.syswrite(["1", "A", sequence].pack("AAN"))
      last_ack = sequence
    end
  end
end

server = TCPServer.new(1234)
sslContext = OpenSSL::SSL::SSLContext.new
sslContext.cert = OpenSSL::X509::Certificate.new(File.read("/tmp/server.crt"))
sslContext.key = OpenSSL::PKey::RSA.new(File.read("/tmp/server.key"), "asdf")
sslServer = OpenSSL::SSL::SSLServer.new(server, sslContext)

while true
  Thread.new(sslServer.accept) { |fd| handle(fd) }
end


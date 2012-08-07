#!/usr/bin/env ruby

require "socket"
require "thread"
require "openssl"

#Thread.abort_on_exception = true

def handle(fd)
  puts fd
  last_ack = 0
  window_size = 0

  while true
    version = fd.read(1)
    #puts "version: #{version}"
    frame = fd.read(1)
    #puts "frame: #{frame}"

    if frame == "W" # window size
      window_size = fd.read(4).unpack("N").first
      puts "Window size: #{window_size}"
      next
    elsif frame != "D"
      puts "Unexpected frame type: #{frame}"
      fd.close
      return
    end

    # data frame
    sequence = fd.read(4).unpack("N").first
    #puts "seq: #{sequence}"
    count = fd.read(4).unpack("N").first
    #puts "count: #{count}"

    map = {}
    count.times do 
      key_len = fd.read(4).unpack("N").first
      key = fd.read(key_len);
      value_len = fd.read(4).unpack("N").first
      value = fd.read(value_len);
      map[key] = value
    end
    #p sequence => map

    #puts "v: #{sequence - last_ack} vs #{window_size}"
    if sequence - last_ack >= window_size
      #fd.close; return;

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

puts "OK"
while true
  begin 
    Thread.new(sslServer.accept) do |fd| 
      begin
        handle(fd) 
      rescue => e
        p e
        raise e
      end
    end
  rescue => e
    p e
  end
end

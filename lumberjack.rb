#!/usr/bin/env ruby

require "socket"
require "thread"
require "openssl"
require "zlib"
#require "lz4-ruby"

class IOWrap
  def initialize(io)
    @io = io
    @buffer = ""
  end

  def read(bytes)
    if @buffer.empty?
      #puts "reading direct from @io"
      return @io.read(bytes)
    elsif @buffer.length > bytes
      #puts "reading buffered"
      data = @buffer[0...bytes]
      @buffer[0...bytes] = ""
      return data
    else
      data = @buffer.clone
      @buffer.clear
      return data + @io.read(bytes - data.length)
    end
  end

  def pushback(data)
    #puts "Pushback: #{data[0..30].inspect}..."
    @buffer += data
  end

  def method_missing(method, *args)
    @io.send(method, *args)
  end

end

def handle(fd)
  last_ack = 0
  window_size = 0

  io = IOWrap.new(fd)

  data_frames = 0
  while true
    version = io.read(1)
    #puts "version: #{version.inspect}"
    frame = io.read(1)
    #puts "frame: #{frame.inspect}"

    if frame == "W" # window size
      window_size = io.read(4).unpack("N").first / 2
      #puts "Window size: #{window_size}"
      next
    elsif frame == "C" # compressed data
      length = io.read(4).unpack("N").first
      #puts "Compressed frame length #{length}"
      compressed = io.read(length)
      original = Zlib::Inflate.inflate(compressed)
      #original = LZ4::uncompress(compressed, length)
      io.pushback(original)
      next
    elsif frame != "D"
      #puts "Unexpected frame type: #{version.inspect} / #{frame.inspect}"
      io.close
      return
    end
    #
    # data frame
    sequence = io.read(4).unpack("N").first
    #puts "seq: #{sequence}"
    count = io.read(4).unpack("N").first
    #puts "count: #{count}"

    map = {}
    count.times do 
      key_len = io.read(4).unpack("N").first
      key = io.read(key_len);
      value_len = io.read(4).unpack("N").first
      value = io.read(value_len);
      map[key] = value
    end
    #p sequence => map
    #sleep 0.1
    data_frames += 1
    if data_frames % 10000 == 0
      p :data_frames => data_frames 
      p map
    end


    #puts "v: #{sequence - last_ack} vs #{window_size}"
    if sequence - last_ack >= window_size
      #fd.close; return;

      # ack this.
      io.syswrite(["1", "A", sequence].pack("AAN"))
      last_ack = sequence
    end
  end
end

server = TCPServer.new(5001)
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
        puts "handle() exception: #{e}"
        puts e.backtrace
        raise e
      end
    end
  rescue => e
    p e
  end
end

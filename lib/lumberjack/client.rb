require "socket"
require "thread"
require "openssl"
require "zlib"

module Lumberjack
  
  WINDOW_SIZE = 60000
  SEQUENCE_MAX = (2**(0.size * 8 -2) -1)

  class Client

  end

  class Socket

    # Create a new Lumberjack Socket.
    #
    # - options is a hash. Valid options are:
    #
    # * :port - the port to listen on
    # * :address - the host/address to bind to
    # * :ssl_certificate - the path to the ssl cert to use
    attr_reader :sequence
    attr_reader :sent

    def initialize(opts={})
      @sequence = 0
      @last_ack = 0
      @sent = 0
      @opts = {
        :port => 0,
        :address => "127.0.0.1",
        :ssl_certificate => nil,
      }.merge(opts)

      tcp_socket = TCPSocket.new(@opts[:address], @opts[:port])
      openssl_cert = OpenSSL::X509::Certificate.new(File.read(@opts[:ssl_certificate]))
      @socket = OpenSSL::SSL::SSLSocket.new(tcp_socket)
      @socket.connect

      #if @socket.peer_cert.to_s != openssl_cert.to_s
      #  raise "Client and server certificates do not match."
      #end

      @socket.syswrite(["1", "W", Lumberjack::WINDOW_SIZE].pack("AAN"))
    end

    private 
    def inc
      @sequence = 0 if @sequence > Lumberjack::SEQUENCE_MAX
      @sequence = @sequence + 1
    end

    private
    def write(msg)
      @socket.syswrite(msg)
    end

    public
    def write_hash(hash)
      frame = to_frame(hash, inc)
      ack if (@sequence - @last_ack) >= Lumberjack::WINDOW_SIZE
      write(frame)
    end

    private
    def ack
      version = @socket.read(1)
      type = @socket.read(1)
      raise "Whoa we shouldn't get this frame: #{type}" if type != "A"
      @last_ack = @socket.read(4).unpack("N").first
      ack if (@sequence - @last_ack) >= Lumberjack::WINDOW_SIZE
    end

    private
    def to_frame(hash, sequence)
      frame = ["1", "D", sequence]
      pack = "AAN"
      keys = deep_keys(hash)
      frame << keys.length
      pack << "N"
      keys.each do |k|
        val = deep_get(hash,k)
        key_length = k.length
        val_length = val.length
        frame << key_length
        pack << "N"
        frame << k
        pack << "A#{key_length}"
        frame << val_length
        pack << "N"
        frame << val
        pack << "A#{val_length}"
      end
      frame.pack(pack)
    end

    private
    def deep_get(hash, key="")
      return hash if key.nil?
      deep_get(
        hash[key.split('.').first],
        key[key.split('.').first.length+1..key.length]
      )
    end

    private
    def deep_keys(hash, prefix="")
      keys = []
      hash.each do |k,v|
        keys << "#{prefix}#{k}" if v.class == String
        keys << hash[k].deep_keys(hash, "#{k}.") if v.class == Hash
      end
      keys.flatten
    end
  end
end
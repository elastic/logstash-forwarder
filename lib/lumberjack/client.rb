# encoding: utf-8
require "socket"
require "thread"
require "openssl"
require "zlib"

module Lumberjack

  SEQUENCE_MAX = (2**32-1).freeze

  class Client
    def initialize(opts={})
      @opts = {
        :port => 0,
        :addresses => [],
        :ssl_certificate => nil,
        :window_size => 5000
      }.merge(opts)

      @opts[:addresses] = [@opts[:addresses]] if @opts[:addresses].class == String
      raise "Must set a port." if @opts[:port] == 0
      raise "Must set atleast one address" if @opts[:addresses].empty? == 0
      raise "Must set a ssl certificate or path" if @opts[:ssl_certificate].nil?

      @socket = connect

    end

    private
    def connect
      addrs = @opts[:addresses].shuffle
      begin
        raise "Could not connect to any hosts" if addrs.empty?
        opts = @opts
        opts[:address] = addrs.pop
        Lumberjack::Socket.new(opts)
      rescue *[Errno::ECONNREFUSED,SocketError]
        retry
      end
    end

    public
    def write(hash)
      @socket.write_hash(hash)
    end

    public
    def host
      @socket.host
    end
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
    attr_reader :window_size
    attr_reader :host
    def initialize(opts={})
      @sequence = 0
      @last_ack = 0
      @opts = {
        :port => 0,
        :address => "127.0.0.1",
        :ssl_certificate => nil,
        :window_size => 5000
      }.merge(opts)
      @host = @opts[:address]
      @window_size = @opts[:window_size]

      connection_start(opts)
    end

    private
    def connection_start(opts)
      tcp_socket = TCPSocket.new(opts[:address], opts[:port])
      @socket = OpenSSL::SSL::SSLSocket.new(tcp_socket)
      @socket.connect
      @socket.syswrite(["1", "W", @window_size].pack("AAN"))
    end

    private 
    def inc
      @sequence = 0 if @sequence + 1 > Lumberjack::SEQUENCE_MAX
      @sequence = @sequence + 1
    end

    private
    def write(msg)
      compress = Zlib::Deflate.deflate(msg)
      payload = ["1","C",compress.length,compress].pack("AANA#{compress.length}")
      # SSLSocket has a limit of 16k per message
      # execute multiple writes if needed
      bytes_written = 0
      while bytes_written < payload.bytesize
        bytes_written += @socket.syswrite(payload.byteslice(bytes_written..-1))
      end
    end

    public
    def write_hash(hash)
      frame = Encoder.to_compressed_frame(hash, inc)
      ack if unacked_sequence_size >= @window_size
      write frame
    end

    private
    def ack
      _, type = read_version_and_type
      raise "Whoa we shouldn't get this frame: #{type}" if type != "A"
      @last_ack = read_last_ack
      ack if unacked_sequence_size >= @window_size
    end

    private
    def unacked_sequence_size
      sequence - (@last_ack + 1)
    end

    private
    def read_version_and_type
      version = @socket.read(1)
      type    = @socket.read(1)
      [version, type]
    end
    private
    def read_last_ack
      @socket.read(4).unpack("N").first
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
        key_length = k.bytesize
        val_length = val.bytesize
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
        keys << deep_keys(hash[k], "#{k}.") if v.class == Hash
      end
      keys.flatten
    end
  end

  module Encoder
    def self.to_compressed_frame(hash, sequence)
      compress = Zlib::Deflate.deflate(to_frame(hash, sequence))
      ["1", "C", compress.bytesize, compress].pack("AANA#{compress.length}")
    end

    def self.to_frame(hash, sequence)
      frame = ["1", "D", sequence]
      pack = "AAN"
      keys = deep_keys(hash)
      frame << keys.length
      pack << "N"
      keys.each do |k|
        val = deep_get(hash,k)
        key_length = k.bytesize
        val_length = val.bytesize
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
    def self.deep_get(hash, key="")
      return hash if key.nil?
      deep_get(
        hash[key.split('.').first],
        key[key.split('.').first.length+1..key.length]
      )
    end
    private
    def self.deep_keys(hash, prefix="")
      keys = []
      hash.each do |k,v|
        keys << "#{prefix}#{k}" if v.class == String
        keys << deep_keys(hash[k], "#{k}.") if v.class == Hash
      end
      keys.flatten
    end
  end # module Encoder
end

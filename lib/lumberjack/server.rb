require "socket"
require "thread"
require "openssl"
require "zlib"

module Lumberjack
  class Server
    attr_reader :port

    # Create a new Lumberjack server.
    #
    # - options is a hash. Valid options are:
    #
    # * :port - the port to listen on
    # * :address - the host/address to bind to
    # * :ssl_certificate - the path to the ssl cert to use
    # * :ssl_key - the path to the ssl key to use
    # * :ssl_key_passphrase - the key passphrase (optional)
    def initialize(options={})
      @options = {
        :port => 0,
        :address => "0.0.0.0",
        :ssl_certificate => nil,
        :ssl_key => nil,
        :ssl_key_passphrase => nil,
      }.merge(options)

      [:ssl_certificate, :ssl_key].each do |k|
        if @options[k].nil?
          raise "You must specify #{k} in Lumberjack::Server.new(...)"
        end
      end

      @tcp_server = TCPServer.new(@options[:port])
      # Query the port in case the port number is '0'
      # TCPServer#addr == [ address_family, port, address, address ]
      @port = @tcp_server.addr[1]
      @ssl = OpenSSL::SSL::SSLContext.new
      @ssl.cert = OpenSSL::X509::Certificate.new(File.read(@options[:ssl_certificate]))
      @ssl.key = OpenSSL::PKey::RSA.new(File.read(@options[:ssl_key]),
                                        @options[:ssl_key_passphrase])
      @ssl_server = OpenSSL::SSL::SSLServer.new(@tcp_server, @ssl)
    end # def initialize

    def run(&block)
      while true
        begin 
          Thread.new(@ssl_server.accept) do |fd| 
            Connection.new(fd).run(&block)
          end
        rescue => e
          p :accept_error => e
        end
      end
    end # def run
  end # class Server

  class Connection
    def initialize(fd)
      @fd = fd
    end # def initialize

    def run(&block)
      each_event(&block)
    end # def run

    def each_event(&block)
      last_ack = 0
      window_size = 0
      io = IOWrap.new(@fd)
      while true
        version = io.read(1)
        frame = io.read(1)

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
        count = io.read(4).unpack("N").first

        map = {}
        count.times do 
          key_len = io.read(4).unpack("N").first
          key = io.read(key_len)
          value_len = io.read(4).unpack("N").first
          value = io.read(value_len)
          map[key] = value
        end

        block.call(map)

        if sequence - last_ack >= window_size
          # ack this.
          io.syswrite(["1", "A", sequence].pack("AAN"))
          last_ack = sequence
        end
      end
    end # def each_event
  end # class Connection

  # Wrap an io-like object but support pushback.
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
        data = @buffer.slice!(0...bytes)
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
  end # class IOWrap
end # module Lumberjack


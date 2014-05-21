require "socket"
require "thread"
require "timeout"
require "openssl"
require "zlib"

module Lumberjack
  class ShutdownSignal < StandardError; end
  class ProtocolError < StandardError; end

  # This allows us to catch a reference to the accepted socket
  # So that if the SSL layer throws an error, we can grab it to get the peer address that caused it
  class ExtendedTCPServer < TCPServer
    attr_reader :last_peer

    def accept()
      sock = super()
      peer = sock.peeraddr(:numeric)
      @last_peer = "#{peer[2]}:#{peer[1]}"
      return sock
    end
  end

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
        :logger => nil,
      }.merge(options)

      @logger = @options[:logger]

      [:ssl_certificate, :ssl_key].each do |k|
        if @options[k].nil?
          raise "You must specify #{k} in Lumberjack::Server.new(...)"
        end
      end

      @tcp_server = ExtendedTCPServer.new(@options[:port])
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
      event_queue = SizedQueue.new(500)
      spooler_thread = nil
      client_threads = Hash.new
      ack_resume = Hash.new
      ack_resume_mutex = Mutex.new

      begin
        # Why a spooler thread? Well we don't know what &block is! We want connection threads to be non-blocking so they DON'T timeout
        # Non-blocking means we can keep clients informed of progress, and response in a timely fashion. We could create this with
        # a timeout wrapper around the &block call but we'd then be generating exceptions in someone else's code
        # So we allow the caller to block us - but only our spooler thread - our other threads are safe and we can use timeout
        spooler_thread = Thread.new do
          begin
            while true
              block.call(event_queue.pop)
            end
          rescue ShutdownSignal
            # Flush whatever we have left
          end
          while event_queue.length
            block.call(event_queue.pop)
          end
        end

        while true
          # NOTE: This means ssl accepting is single-threaded.
          begin
            client = @ssl_server.accept
          rescue EOFError, OpenSSL::SSL::SSLError, IOError => e
            # ssl handshake failure or other issue, skip it.
            @logger.warn("[Lumberjack] Connection from #{@tcp_server.last_peer} failed to initialise: #{e}") if not @logger.nil?
            client.close rescue nil
            next
          end

          @logger.info("[Lumberjack] New connection from #{@tcp_server.last_peer}") if not @logger.nil?

          # Clear up finished threads
          client_threads.delete_if do |k, thr|
            not thr.alive?
          end

          # Start a new connection thread
          client_threads[client] = Thread.new(client, @tcp_server.last_peer) do |client, peer|
            Connection.new(@logger, client, peer).run(event_queue)
          end

          # Reset client so if ssl_server.accept fails, we don't close the previous connection in client since it was left untouched
          client = nil
        end
      ensure
        # Raise shutdown in all client threads and join then
        client_threads.each do |thr|
          thr.raise(ShutdownSignal)
        end
        client_threads.each(&:join)

        # Signal the spooler thread to stop
        if not spooler_thread.nil?
          spooler_thread.raise(ShutdownSignal)
          spooler_thread.join
        end
      end # ensure
    end # def run
  end # class Server

  class Parser
    def initialize
      @buffer_offset = 0
      @buffer = ""
      @buffer.force_encoding("BINARY")
      transition(:header, 2)
    end # def initialize

    def transition(state, next_length)
      @state = state
      #puts :transition => state
      # TODO(sissel): Assert this self.respond_to?(state)
      # TODO(sissel): Assert state is in STATES
      # TODO(sissel): Assert next_length is a number
      need(next_length)
    end # def transition

    # Feed data to this parser.
    # 
    # Currently, it will return the raw payload of websocket messages.
    # Otherwise, it returns nil if no complete message has yet been consumed.
    #
    # @param [String] the string data to feed into the parser. 
    # @return [String, nil] the websocket message payload, if any, nil otherwise.
    def feed(data, &block)
      @buffer << data
      #p :need => @need
      while have?(@need)
        send(@state, &block) 
        #case @state
          #when :header; header(&block)
          #when :window_size; window_size(&block)
          #when :data_lead; data_lead(&block)
          #when :data_field_key_len; data_field_key_len(&block)
          #when :data_field_key; data_field_key(&block)
          #when :data_field_value_len; data_field_value_len(&block)
          #when :data_field_value; data_field_value(&block)
          #when :data_field_value; data_field_value(&block)
          #when :compressed_lead; compressed_lead(&block)
          #when :compressed_payload; compressed_payload(&block)
        #end # case @state
      end
      return nil
    end # def <<

    # Do we have at least 'length' bytes in the buffer?
    def have?(length)
      return length <= (@buffer.size - @buffer_offset)
    end # def have?

    # Get 'length' string from the buffer.
    def get(length=nil)
      length = @need if length.nil?
      data = @buffer[@buffer_offset ... @buffer_offset + length]
      @buffer_offset += length
      if @buffer_offset > 16384
        @buffer = @buffer[@buffer_offset  .. -1]
        @buffer_offset = 0
      end
      return data
    end # def get

    # Set the minimum number of bytes we need in the buffer for the next read.
    def need(length)
      @need = length
    end # def need

    FRAME_WINDOW = "W".ord
    FRAME_DATA = "D".ord
    FRAME_COMPRESSED = "C".ord
    def header(&block)
      @frame_version, frame_type = get.bytes.to_a[0..1]
      case frame_type
        when FRAME_WINDOW; transition(:window_size, 4)
        when FRAME_DATA; transition(:data_lead, 8)
        when FRAME_COMPRESSED; transition(:compressed_lead, 4)
        else; raise ProtocolError
      end
    end

    def window_size(&block)
      @window_size = get.unpack("N").first
      transition(:header, 2)
      yield :window_size, @window_size
    end # def window_size

    def data_lead(&block)
      @sequence, @data_count = get.unpack("NN")
      @data = {}
      transition(:data_field_key_len, 4)
      yield :data_lead
    end

    def data_field_key_len(&block)
      key_len = get.unpack("N").first
      transition(:data_field_key, key_len)
    end

    def data_field_key(&block)
      @key = get
      transition(:data_field_value_len, 4)
    end

    def data_field_value_len(&block)
      transition(:data_field_value, get.unpack("N").first)
    end

    def data_field_value(&block)
      @value = get

      @data_count -= 1
      @data[@key] = @value

      if @data_count > 0
        transition(:data_field_key_len, 4)
      else
        # emit the whole map now that we found the end of the data fields list.
        yield :data, @frame_version, @sequence, @data
        transition(:header, 2)
      end

    end # def data_field_value

    def compressed_lead(&block)
      length = get.unpack("N").first
      transition(:compressed_payload, length)
    end
    
    def compressed_payload(&block)
      original = Zlib::Inflate.inflate(get)
      transition(:header, 2)

      # Parse the uncompressed payload.
      feed(original, &block)
    end
  end # class Parser

  class Connection
    def initialize(logger, fd, peer)
      super()
      @parser = Parser.new
      @logger = logger
      @fd = fd
      @peer = peer
      @last_window_ack = nil

      # Safe defaults until we are told by the client
      @window_size = 1

      reset_timeout
    end

    def run(event_queue)
      while true
        begin
          # If we don't receive anything after the main timeout - something is probably wrong
          buffer = Timeout::timeout(@timeout - Time.now.to_i) do
            buffer = @fd.sysread(16384)
            next buffer
          end
        rescue Timeout::Error
          # TODO(driskell): Should we disconnect? Or keep waiting?
          # Clients should really ping to keep this connection open
          # So we don't start increasing the reconnect on old clients we can just wait for now
          reset_timeout
          next
        end
        @parser.feed(buffer) do |event, *args|
          case event
            when :window_size; window_size(*args)
            when :data_lead; data_lead()
            when :data; data(*args, event_queue)
          end
          #send(event, *args)
        end # feed
      end # while true
    rescue EOFError, OpenSSL::SSL::SSLError, IOError, Errno::ECONNRESET => e
      # EOF or other read errors, only action is to shutdown which we'll do in
      # 'ensure'
      @logger.warn("[Lumberjack] SSL error on connection from #{@peer}: #{e}") if not @logger.nil?
    rescue ProtocolError
      # Connection abort request due to a protocol error
      @logger.warn("[Lumberjack] Protocol error on connection from #{@peer}") if not @logger.nil?
    rescue => e
      # Some other unknown problem
      @logger.warn("[Lumberjack] Unknown error on connection from #{@peer}: #{e}") if not @logger.nil?
    ensure
      # Try to ensure it's closed, but if this fails I don't care.
      @logger.info("[Lumberjack] Closing connection from #{@peer}") if not @logger.nil?
      @fd.close rescue nil
    end # def run

    def reset_timeout()
      @timeout = Time.now.to_i + 1800
    end

    def reset_ack_timeout()
      @ack_timeout = Time.now.to_i + 5
      reset_timeout
    end

    def window_size(size)
      @window_size = size
    end

    def data_lead()
      reset_ack_timeout
    end

    def data(frame_version, sequence, map, event_queue)
      # If our current last_window_sequence is 0, this is a new connection
      # However, the client doesn't necessarily want to start from 0... so populate initial last_window_sequence with first-1
      if @last_window_ack.nil?
        @last_window_ack = sequence - 1
      end

      while true
        begin
          # Follow the ack timeout here - this needs to be smaller
          Timeout::timeout(@ack_timeout - Time.now.to_i) do
            event_queue << map
          end
        rescue Timeout::Error
          # While we're busy - keeping sending partial ACKs for the last sequence we finished
          # But ONLY if we're frame version 2+ so we don't break the old clients that don't support partial ACKs
          if frame_version > 1
            send_ack(sequence - 1)
          else
            reset_ack_timeout
          end
        else
          break
        end
      end
      if (sequence - @last_window_ack) >= @window_size
        send_ack(sequence)
        @last_window_ack = sequence
      end
    end

    def send_ack(sequence)
      @fd.syswrite(["1A", sequence].pack("A*N"))
      reset_ack_timeout
    end
  end # class Connection

end # module Lumberjack

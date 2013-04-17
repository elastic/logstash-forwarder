require "ffi-rzmq"
require "zlib"
require "rbnacl"
require "json"
require "stud/try"

module Lumberjack
  class Server2
    # Create a new Lumberjack server.
    #
    # - options is a hash. Valid options are:
    #
    # * :port - the port to listen on
    # * :address - the host/address to bind to
    def initialize(options={})
      @options = {
        :workers => 1,

        # Generate an inproc url for workers to attach to
        :worker_endpoint => "inproc://#{Time.now.to_f}#{rand}"
      }.merge(options)

      [:my_secret_key, :their_public_key, :endpoint].each do |k|
        if @options[k].nil?
          raise "You must specify #{k} in Lumberjack::Server.new(...)"
        end
      end

      @context = ZMQ::Context.new

      @cryptobox = Crypto::Box.new(
        Crypto::PublicKey.new(@options[:their_public_key]),
        Crypto::PrivateKey.new(@options[:my_secret_key]))
    end # def initialize

    def setup_proxy(context)
      # Socket facing clients
      frontend = context.socket(ZMQ::ROUTER)
      rc = frontend.bind(@options[:endpoint])
      if rc < 0
        puts "RC :("
        raise "Unable to bind lumberjack to #{@options[:endpoint]}"
      end
      
      # Socket facing services
      backend = context.socket(ZMQ::DEALER)
      backend.bind(@options[:worker_endpoint])

      @proxy_thread = Thread.new do
        ZMQ::Device.new(ZMQ::QUEUE, frontend, backend)
        raise "The lumberjack proxy died."
      end
    end

    def run(&block)
      #setup_proxy(@context)

      threads = @options[:workers].times.collect do |i|
        Thread.new do
          puts "Starting worker #{i}"
          run_worker(@context, &block)
        end
      end

      threads.each(&:join)
    end

    def run_worker(context, &block)
      socket = context.socket(ZMQ::REP)
      Stud::try(10.times) do
        #rc = socket.connect(@options[:worker_endpoint])
        rc = socket.bind(@options[:endpoint])
        if rc < 0
          raise "connect to #{@options[:worker_endpoint]} failed"
        end
      end

      ciphertext = ""
      ciphertext.force_encoding("BINARY")
      nonce = ""
      nonce.force_encoding("BINARY")
      count = 0
      start = Time.now
      list = []
      while true
        socket.recv_string(nonce)
        socket.recv_string(ciphertext)

        # Decrypt
        #plaintext = @cryptobox.open(nonce, ciphertext)

        # decompress
        #inflated = Zlib::Inflate.inflate(plaintext)

        # JSON
        #events = JSON.parse(inflated)
        #events.each do |event|
          #yield event
        #end

        # TODO(sissel): yield each event
        #count += events.count
        count += 1

        # Reply to acknowledge.
        # Currently there is no response message to put.
        socket.send_string("")

        if count > 100
          puts :rate => (count / (Time.now - start))
          count = 0
          start = Time.now
        end
      end
    end # def run_worker
  end # class Server2
end # module Lumberjack

if __FILE__ == $0
  a = Lumberjack::Server2.new(
    :workers => 1,
    :endpoint => "tcp://127.0.0.1:12345",
    :their_public_key => File.read("../../nacl.public").force_encoding("BINARY"),
    :my_secret_key => File.read("../../nacl.secret").force_encoding("BINARY"))

  count = 0
  #start = Time.now
  #require "thread"
  #q = Queue.new
  #q = java.util.concurrent.LinkedBlockingQueue.new
  #Thread.new { a.run { |e| q.put(e) } }
  a.run { |e| }

  #while q.take
    #count += 1
    #if count > 100000
      #puts count / (Time.now - start)
      #count = 0
      #start = Time.now
    #end
  #end
end

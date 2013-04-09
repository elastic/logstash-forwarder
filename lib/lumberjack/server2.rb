require "ffi-rzmq"
require "zlib"
require "rbnacl"
require "json"

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
        :endpoint => "tcp://0.0.0.0:3333",
        :my_secret_key => nil,
        :their_public_key => nil,
      }.merge(options)

      [:my_secret_key, :their_public_key].each do |k|
        if @options[k].nil?
          raise "You must specify #{k} in Lumberjack::Server.new(...)"
        end
      end

      @context = ZMQ::Context.new
      @socket = @context.socket(ZMQ::REP)
      @socket.bind(@options[:endpoint])

      @cryptobox = Crypto::Box.new(
        Crypto::PublicKey.new(@options[:their_public_key]),
        Crypto::PrivateKey.new(@options[:my_secret_key]))
    end # def initialize

    def run(&block)
      ciphertext = ""
      ciphertext.force_encoding("BINARY")
      nonce = ""
      nonce.force_encoding("BINARY")
      count = 0
      start = Time.now
      while true
        @socket.recv_string(nonce)
        @socket.recv_string(ciphertext)

        # Decrypt
        plaintext = @cryptobox.open(nonce, ciphertext)

        # decompress
        inflated = Zlib::Inflate.inflate(plaintext)

        # JSON
        events = JSON.parse(inflated)

        # TODO(sissel): yield each event
        count += events.count
        @socket.send_string("")
        #count += 4096

        if count > 100000
          puts :rate => (count / (Time.now - start))
          count = 0
          start = Time.now
        end
      end
    end # def run
  end # class Server2
end # module Lumberjack

if __FILE__ == $0
  a = Lumberjack::Server2.new(
    :their_public_key => File.read("../../nacl.public").force_encoding("BINARY"),
    :my_secret_key => File.read("../../nacl.secret").force_encoding("BINARY"))

  a.run do |e|
    p :event => e
  end
end

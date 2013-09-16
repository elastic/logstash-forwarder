$: << File.realpath(File.join(File.dirname(__FILE__), "..", "lib"))
require "tempfile"
require "lumberjack/server"
require "insist"
require "stud/try"
require "json"

describe "lumberjack" do
  before :each do
    # TODO(sissel): Generate a self-signed SSL cert
    @ssl_cert = Tempfile.new("lumberjack-test-file")
    @ssl_key = Tempfile.new("lumberjack-test-file")
    @ssl_csr = Tempfile.new("lumberjack-test-file")
    @config_file = Tempfile.new("lumberjack-test-file")

    # Generate the ssl key
    system("openssl genrsa -out #{@ssl_key.path} 1024")
    system("openssl req -new -key #{@ssl_key.path} -batch -out #{@ssl_csr.path}")
    system("openssl x509 -req -days 365 -in #{@ssl_csr.path} -signkey #{@ssl_key.path} -out #{@ssl_cert.path}")

    @server = Lumberjack::Server.new(
      :ssl_certificate => @ssl_cert.path,
      :ssl_key => @ssl_key.path
    )
    config = { "network" => { "servers"=> [ "localhost:#{@server.port}"], "ssl ca"=>@ssl_cert.path }, "files"=>[{ "paths"=>["-"] }] }
    @config_file.write config.to_json
    @config_file.close
    @lumberjack = IO.popen("build/bin/lumberjack --config #{@config_file.path}", "r+")
    @event_queue = Queue.new
    @server_thread = Thread.new do
      @server.run do |event|
        @event_queue << event
      end
    end
  end # before each

  after :each do
    @ssl_cert.close
    @ssl_key.close
    @ssl_csr.close
    Process::kill("KILL", @lumberjack.pid)
    Process::wait(@lumberjack.pid)
  end

  it "should follow stdin" do
    count = rand(50000) + 2500000
    message = "hello world foo bar baz fizz=lkjwelfkj"
    Thread.new do 
      count.times do |i|
        @lumberjack.puts("#{message} #{i}")

        # random sleep 0.01% of the time
        sleep(rand) if rand < 0.0001
      end
      @lumberjack.close
    end

    # Now verify that we have all the data and in the correct order.
    host = Socket.gethostname
    count.times do |i|
      event = @event_queue.pop
      insist { event["line"] } == "#{message} #{i}"
      #insist { event["file"] } == @file.path
      insist { event["host"] } == host
    end
    insist { @event_queue }.empty?
  end
end

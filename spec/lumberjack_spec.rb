$: << File.realpath(File.join(File.dirname(__FILE__), "..", "lib"))
require "tempfile"
require "lumberjack/server"
require "insist"
require "stud/try"
require 'timeout'

describe "lumberjack" do
  before :each do
    # TODO(sissel): Generate a self-signed SSL cert
    @file = Tempfile.new("lumberjack-test-file")
    @ssl_cert = Tempfile.new("lumberjack-test-file")
    @ssl_key = Tempfile.new("lumberjack-test-file")
    @ssl_csr = Tempfile.new("lumberjack-test-file")

    # Generate the ssl key
    system("openssl genrsa -out #{@ssl_key.path} 1024")
    system("openssl req -new -key #{@ssl_key.path} -batch -out #{@ssl_csr.path}")
    system("openssl x509 -req -days 365 -in #{@ssl_csr.path} -signkey #{@ssl_key.path} -out #{@ssl_cert.path}")

    @server = Lumberjack::Server.new(
      :ssl_certificate => @ssl_cert.path,
      :ssl_key => @ssl_key.path
    )
    @lumberjack = IO.popen("build/bin/lumberjack --host localhost " \
                           "--port #{@server.port} " \
                           "--ssl-ca-path #{@ssl_cert.path} #{@file.path}",
                           "r")

    @event_queue = Queue.new
    @server_thread = Thread.new do
      @server.run do |event|
        @event_queue << event
      end
    end
  end # before each

  after :each do
    @file.close
    @ssl_cert.close
    @ssl_key.close
    @ssl_csr.close
    Process::kill("KILL", @lumberjack.pid)
    Process::wait(@lumberjack.pid)
  end

  it "should follow a file and emit lines as events" do
    sleep 1 # let lumberjack start up.
    count = rand(5000) + 25000
    count.times do |i|
      @file.puts("hello #{i}")
    end
    @file.close

    # Wait for lumberjack to finish publishing data to us.
    Stud::try(20.times) do
      raise "have #{@event_queue.size}, want #{count}" if @event_queue.size < count
    end

    # Now verify that we have all the data and in the correct order.
    insist { @event_queue.size } == count
    host = Socket.gethostname
    count.times do |i|
      event = @event_queue.pop
      insist { event["line"] } == "hello #{i}"
      insist { event["file"] } == @file.path
      insist { event["host"] } == host
    end
    insist { @event_queue }.empty?
  end

  it "should follow a file with long lines and emit partial long lines as events" do
    sleep 5 # let lumberjack start up.
    count = 4 #rand(5000) + 25000
    length = 16384
    count.times do |i|
      @file.puts("hello #{i} #{'q'*length}")
    end
    @file.close

    # Wait for lumberjack to finish publishing data to us.
    Stud::try(20.times) do
      raise "have #{@event_queue.size}, want #{count}" if @event_queue.size < count
    end

    # Now verify that we have all the data and in the correct order.
    insist { @event_queue.size } == (count* 2)-1
    host = Socket.gethostname
    count.times do |i|
      header = "hello #{i}"
      event = @event_queue.pop
      insist { event["line"] } == "#{header} #{'q'*(length-header.length-3)}"
      insist { event["file"] } == @file.path
      insist { event["host"] } == host
      event = @event_queue.pop
      insist { event["line"] } == "#{'q'*(header.length + 2)}"
      insist { event["file"] } == @file.path
      insist { event["host"] } == host
    end
    insist { @event_queue }.empty?
  end

  it "should follow a slowly-updating file and emit lines as events" do
    sleep 5 # let lumberjack start up.
    count = rand(50) + 1000
    count.times do |i|
      @file.puts("fizzle #{i}")

      # Start fast, then go slower after 80% of the events
      if i > (count * 0.8)
        sleep(rand * 0.200) # sleep up to 200ms
      end
    end
    @file.close

    # Wait for lumberjack to finish publishing data to us.
    Stud::try(20.times) do
      raise "have #{@event_queue.size}, want #{count}" if @event_queue.size < count
    end

    # Now verify that we have all the data and in the correct order.
    insist { @event_queue.size } == count
    host = Socket.gethostname
    count.times do |i|
      event = @event_queue.pop
      insist { event["line"] } == "fizzle #{i}"
      insist { event["file"] } == @file.path
      insist { event["host"] } == host
    end
    insist { @event_queue }.empty?
  end
end

$: << File.realpath(File.join(File.dirname(__FILE__), "..", "lib"))
require "tempfile"
require "lumberjack/server"
require "insist"
require "stud/temporary"
require "stud/try"

describe "logstash-forwarder" do
  before :each do
    # TODO(sissel): Generate a self-signed SSL cert
    @file = Stud::Temporary.file("logstash-forwarder-test-file")
    @file2 = Stud::Temporary.file("logstash-forwarder-test-file")
    @config = Stud::Temporary.file("logstash-forwarder-test-file")
    @ssl_cert = Stud::Temporary.file("logstash-forwarder-test-file")
    @ssl_key = Stud::Temporary.file("logstash-forwarder-test-file")
    @ssl_csr = Stud::Temporary.file("logstash-forwarder-test-file")

    # Generate the ssl key
    system("openssl genrsa -out #{@ssl_key.path} 1024")
    system("openssl req -new -key #{@ssl_key.path} -batch -out #{@ssl_csr.path}")
    system("openssl x509 -req -days 365 -in #{@ssl_csr.path} -signkey #{@ssl_key.path} -out #{@ssl_cert.path}")

    @server = Lumberjack::Server.new(
      :ssl_certificate => @ssl_cert.path,
      :ssl_key => @ssl_key.path
    )

    @config.puts(<<-config)
      {
        "network": {
          "servers": [ "localhost:#{@server.port}" ],
          "ssl ca":  "#{@ssl_cert.path}"
        },
        "files": [
          {
            "paths": [ "#{@file.path}" ]
          },
          {
            "paths": [ "#{@file2.path}" ]
          }
        ]
      }
    config
    @config.close

    @event_queue = Queue.new
    @server_thread = Thread.new do
      @server.run do |event|
        @event_queue << event
      end
    end
  end # before each

  after :each do
    shutdown
    [@file, @file2, @config, @ssl_cert, @ssl_key, @ssl_csr].each do |f|
      if not f.closed?
        f.close
      end
      if File.exists?(f.path)
        File.unlink(f.path)
      end
    end
    if File.exists?(".logstash-forwarder.")
      File.unlink(".logstash-forwarder")
    end
  end

  def startup (config="")
    @logstash_forwarder = IO.popen("build/bin/logstash-forwarder -config #{@config.path}" + (config.empty? ? "" : " " + config), "r")
    sleep 1 # let logstash-forwarder start up.
  end # def startup

  def shutdown
    Process::kill("KILL", @logstash_forwarder.pid)
    Process::wait(@logstash_forwarder.pid)
  end # def shutdown

  it "should follow a file from the end and emit lines as events" do
    # Hide 50 lines in the file - this makes sure we start at the end of the file
    initialcount = 50
    initialcount.times do |i|
      @file.puts("test #{i}")
    end
    @file.close

    @file.reopen(@file.path, "a+")

    startup

    count = rand(5000) + 25000
    count.times do |i|
      @file.puts("hello #{i}")
    end
    @file.close

    # Wait for logstash-forwarder to finish publishing data to us.
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

  it "should follow a file from the beginning and emit lines as events" do
    # Hide 50 lines in the file - this makes sure we start at the end of the file
    initialcount = 50
    initialcount.times do |i|
      @file.puts("test #{i}")
    end
    @file.close

    @file.reopen(@file.path, "a+")

    startup "-from-beginning=true"

    count = rand(5000) + 25000
    count.times do |i|
      @file.puts("hello #{i}")
    end
    @file.close

    totalcount = count + initialcount

    # Wait for logstash-forwarder to finish publishing data to us.
    Stud::try(20.times) do
      raise "have #{@event_queue.size}, want #{totalcount}" if @event_queue.size < totalcount
    end

    # Now verify that we have all the data and in the correct order.
    insist { @event_queue.size } == totalcount
    host = Socket.gethostname
    initialcount.times do |i|
      event = @event_queue.pop
      insist { event["line"] } == "test #{i}"
      insist { event["file"] } == @file.path
      insist { event["host"] } == host
    end
    count.times do |i|
      event = @event_queue.pop
      insist { event["line"] } == "hello #{i}"
      insist { event["file"] } == @file.path
      insist { event["host"] } == host
    end
    insist { @event_queue }.empty?
  end

  it "should follow a slowly-updating file and emit lines as events" do
    startup

    count = rand(50) + 1000
    count.times do |i|
      @file.puts("fizzle #{i}")

      # Start fast, then go slower after 80% of the events
      if i > (count * 0.8)
        @file.flush # So we don't get stupid delays
        sleep(rand * 0.200) # sleep up to 200ms
      end
    end
    @file.close

    # Wait for logstash-forwarder to finish publishing data to us.
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

  it "should follow multiple file, and when restarted, resume them" do
    startup

    finish = false
    while true
      count = rand(2500) + 12500
      totalcount = count * 2
      count.times do |i|
        if finish
          i += count # So the second set of lines have unique numbers
        end
        @file.puts("hello #{i}")
        @file2.puts("hello #{i}")
      end
      @file.close
      @file2.close

      # Wait for logstash-forwarder to finish publishing data to us.
      Stud::try(20.times) do
        raise "have #{@event_queue.size}, want #{totalcount}" if @event_queue.size < totalcount
      end

      # Now verify that we have all the data and in the correct order.
      insist { @event_queue.size } == totalcount
      host = Socket.gethostname
      if finish
        count1 = count
        count2 = count
      else
        count1 = 0
        count2 = 0
      end
      totalcount.times do |i|
        event = @event_queue.pop
        if event["file"] == @file.path
          insist { event["line"] } == "hello #{count1}"
          count1 += 1
        else
          insist { event["file"] } == @file2.path
          insist { event["line"] } == "hello #{count2}"
          count2 += 1
        end
        insist { event["host"] } == host
      end
      insist { @event_queue }.empty?

      break if finish

      # Now restart logstash
      shutdown

      # Reopen the files for more output
      @file.reopen(@file.path, "a+")
      @file2.reopen(@file2.path, "a+")

      # From beginning makes testing this easier - without it we'd need to create lines inbetween shutdown and start and verify them which is more work
      startup "-from-beginning=true"
      sleep(1) # let logstash-forwarder start up

      finish = true
    end
  end

  it "should start newly created files found after startup from beginning and not the end" do
    @file2.close
    File.unlink(@file2.path)

    startup

    count = rand(2500) + 12500
    totalcount = count * 2
    count.times do |i|
      @file.puts("hello #{i}")
    end
    @file.close

    sleep(2)

    FileUtils.cp(@file.path, @file2.path)

    # Wait for logstash-forwarder to finish publishing data to us.
    Stud::try(20.times) do
      raise "have #{@event_queue.size}, want #{totalcount}" if @event_queue.size < totalcount
    end

    # Now verify that we have all the data and in the correct order.
    insist { @event_queue.size } == totalcount
    host = Socket.gethostname
    count1 = 0
    count2 = 0
    totalcount.times do |i|
      event = @event_queue.pop
      if event["file"] == @file.path
        insist { event["line"] } == "hello #{count1}"
        count1 += 1
      else
        insist { event["file"] } == @file2.path
        insist { event["line"] } == "hello #{count2}"
        count2 += 1
      end
      insist { event["host"] } == host
    end
    insist { @event_queue }.empty?
  end

  it "should handle delayed new lines past eof_timeout and emit lines as events" do
    startup

    count = rand(50) + 1000
    count.times do |i|
      if (i + 100) % (count / 2) == 0
        # Make 2 events where we pause for >10s before adding new line, this takes us past eof_timeout
        @file.write("fizzle")
        @file.flush
        sleep(15)
        @file.write(" #{i}\n")
      else
        @file.puts("fizzle #{i}")
      end
    end
    @file.close

    # Wait for logstash-forwarder to finish publishing data to us.
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

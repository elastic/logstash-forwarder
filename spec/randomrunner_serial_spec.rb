$: << File.realpath(File.join(File.dirname(__FILE__), "..", "lib"))
require "tempfile"
require "lumberjack/server"
require "insist"
require "stud/temporary"
require "stud/try"
require "json"

describe "logstash-forwarder" do

  before :each do
    # TODO(sissel): Generate a self-signed SSL cert

    @random_seed = ARGV.delete('-s')
    if @random_seed == nil
      @random_seed = Random.new_seed
    end

    puts "Using Seed: #{@random_seed}"
    @random = Random.new(@random_seed)

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

    @files = []
    @active_files = []
    @actual_events = []

    @server_thread = Thread.new do
      @server.run do |event|
        @actual_events << event
      end
    end
  end

  after :each do
    shutdown
    [@config, @ssl_cert, @ssl_key, @ssl_csr].each do |f|
      if not f.closed?
        f.close
      end
      if File.exists?(f.path)
        File.unlink(f.path)
      end
    end

    @active_files.each do |f|
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
    sleep 10 # let logstash-forwarder start up.
  end

  def shutdown
    Process::kill("KILL", @logstash_forwarder.pid)
    Process::wait(@logstash_forwarder.pid)
  end

  def generate_event(probabilities={:emit => 0.999, :rotate => 0.001})
    r = @random.rand(0.0..1.0)

    probabilities.each_pair do |key, value|
      if r < value
        return key
      end

      r -= value
    end
  end

  def ensure_delete(file)
    @active_files << file
  end

  def rotate_file(file, index)
    path = file.path
    rotated_path = path + "_" +SecureRandom.uuid
    puts "\nRotating file: #{File.basename(path)}"
    File.rename(path, rotated_path)
    file.close
    ensure_delete(file)

    @files[index] = File.new(path, "a+")

    ensure_delete(@files[index])

    sleep(15)
  end


  it "should follow multiple explicit files from the end, no rotations, sequential writes" do

    @files = Array.new(@random.rand(1..20)){ |i| Stud::Temporary.file("logstash-forwarder-test-file") }
    json = {
        :network => {
            :servers => ["localhost:#{@server.port}"],
            "ssl ca" => "#{@ssl_cert.path}",
        },
        :files => @files.map do |u|
          { :paths => [u.path] }
        end
    }

    @config.puts(json.to_json)
    @config.close

    # initialize files with 1-100 lines of text
    @files.each do |file|
      ensure_delete(file)
      initial = @random.rand(1..100)
      initial.times do |count|
        file.puts("initial #{count}")
      end
      file.close
      file.reopen(file.path, "a+")
    end

    # Start LSF
    startup

    expected_events = Array.new
    count = @random.rand(10000..25000)
    count.times do |i|
      selected_file = @files[@random.rand(@files.length)]
      selected_file.puts("#{i}")
      expected_events << {
          "line" => "#{i}",
          "file" => selected_file.path,
          "host" => Socket.gethostname
      }
    end

    @files.each do |f|
      f.close
    end

    # Wait for logstash-forwarder to finish publishing data to us.
    Stud::try(30.times) do
      raise "have #{@actual_events.size}, want #{count}" if @actual_events.size < count
    end

    # events are not guaranteed to arrive in-order, so sort by line value
    @actual_events = @actual_events.sort_by{ |hsh| Integer(hsh["line"]) }

    # Now verify that we have all the data
    insist { @actual_events.size } == count
    @actual_events.each_index do |index|
      ["line", "file", "host"].each do |property|
        insist { @actual_events[index][property] } == expected_events[index][property]
      end
    end
  end #end test


  it "should follow multiple files via glob from the end, no rotations, sequential writes" do

    @files = Array.new(@random.rand(1..20)){ |i| Stud::Temporary.file("logstash-forwarder-test-file") }
    dir = File.dirname(@files[0])

    json = {
        :network => {
            :servers => ["localhost:#{@server.port}"],
            "ssl ca" => "#{@ssl_cert.path}",
        },
        :files => [
            :paths => ["#{dir}/logstash-forwarder-test-file-*"]
        ]
    }

    @config.puts(json.to_json)
    @config.close

    # initialize files with 1-100 lines of text
    @files.each do |file|
      ensure_delete(file)
      initial = @random.rand(1..100)
      initial.times do |count|
        file.puts("initial #{count}")
      end
      file.close
      file.reopen(file.path, "a+")
    end

    # Start LSF
    startup

    expected_events = Array.new
    count = @random.rand(10000..25000)
    count.times do |i|
      selected_file = @files[@random.rand(@files.length)]
      selected_file.puts("#{i}")
      expected_events << {
          "line" => "#{i}",
          "file" => selected_file.path,
          "host" => Socket.gethostname
      }
    end

    @files.each do |f|
      f.close
    end

    # Wait for logstash-forwarder to finish publishing data to us.
    Stud::try(30.times) do
      raise "have #{@actual_events.size}, want #{count}" if @actual_events.size < count
    end

    # events are not guaranteed to arrive in-order, so sort by line value
    @actual_events = @actual_events.sort_by{ |hsh| Integer(hsh["line"]) }

    # Now verify that we have all the data
    insist { @actual_events.size } == count
    @actual_events.each_index do |index|
      ["line", "file", "host"].each do |property|
        insist { @actual_events[index][property] } == expected_events[index][property]
      end
    end
  end #end test

  it "should follow multiple explicit files from the end, with rotations, sequential writes" do

    @files = Array.new(@random.rand(1..20)){ |i| Stud::Temporary.file("logstash-forwarder-test-file") }
    json = {
        :network => {
            :servers => ["localhost:#{@server.port}"],
            "ssl ca" => "#{@ssl_cert.path}",
        },
        :files => @files.map do |u|
          { :paths => [u.path] }
        end
    }

    @config.puts(json.to_json)
    @config.close

    # initialize files with 1-100 lines of text
    @files.each do |file|
      ensure_delete(file)
      initial = @random.rand(1..100)
      initial.times do |count|
        file.puts("initial #{count}")
      end
      file.close
      file.reopen(file.path, "a+")
    end

    # Start LSF
    startup

    expected_events = Array.new
    count = @random.rand(5000..10000)

    count.times do |i|
      selected_file_index = @random.rand(@files.length)
      selected_file = @files[selected_file_index]

      case generate_event
        when :emit
          selected_file.puts("#{i}")
          expected_events << {
              "line" => "#{i}",
              "file" => selected_file.path,
              "host" => Socket.gethostname
          }
          print "."
          $stdout.flush
        when :rotate
          rotate_file(selected_file, selected_file_index)
      end
    end

    @files.each do |f|
      f.close
    end

    # Wait for logstash-forwarder to finish publishing data to us.
    Stud::try(100.times) do
      raise "have #{@actual_events.size}, want #{expected_events.length}" if @actual_events.size < expected_events.length
    end

    # events are not guaranteed to arrive in-order, so sort by line value
    @actual_events = @actual_events.sort_by{ |hsh| Integer(hsh["line"]) }

    # Now verify that we have all the data
    insist { @actual_events.size } == expected_events.length
    @actual_events.each_index do |index|
      ["line", "file", "host"].each do |property|
        insist { @actual_events[index][property] } == expected_events[index][property]
      end
    end
  end #end test


  it "should follow multiple files via glob from the end, with rotations, sequential writes" do

    @files = Array.new(@random.rand(1..20)){ |i| Stud::Temporary.file("logstash-forwarder-test-file") }
    dir = File.dirname(@files[0])

    json = {
        :network => {
            :servers => ["localhost:#{@server.port}"],
            "ssl ca" => "#{@ssl_cert.path}",
        },
        :files => [
            :paths => ["#{dir}/logstash-forwarder-test-file-*"]
        ]
    }

    @config.puts(json.to_json)
    @config.close

    # initialize files with 1-100 lines of text
    @files.each do |file|
      ensure_delete(file)
      initial = @random.rand(1..100)
      initial.times do |count|
        file.puts("initial #{count}")
      end
      file.close
      file.reopen(file.path, "a+")
    end

    # Start LSF
    startup

    expected_events = Array.new
    count = @random.rand(5000..10000)

    count.times do |i|
      selected_file_index = @random.rand(@files.length)
      selected_file = @files[selected_file_index]

      case generate_event()
        when :emit
          selected_file.puts("#{i}")
          expected_events << {
              "line" => "#{i}",
              "file" => selected_file.path,
              "host" => Socket.gethostname
          }
          print "."
          $stdout.flush
        when :rotate
          rotate_file(selected_file, selected_file_index)
      end
    end

    @files.each do |f|
      f.close
    end

    # Wait for logstash-forwarder to finish publishing data to us.
    Stud::try(100.times) do
      raise "have #{@actual_events.size}, want #{expected_events.length}" if @actual_events.size < expected_events.length
    end

    # events are not guaranteed to arrive in-order, so sort by line value
    @actual_events = @actual_events.sort_by{ |hsh| Integer(hsh["line"]) }

    # Now verify that we have all the data
    insist { @actual_events.size } == expected_events.length
    @actual_events.each_index do |index|
      ["line", "file", "host"].each do |property|
        insist { @actual_events[index][property] } == expected_events[index][property]
      end
    end
  end #end test


  it "should follow multiple explicit files from end, new files from beginning, no rotations, sequential writes" do

    @files = Array.new(@random.rand(1..5)){ |i| Stud::Temporary.file("logstash-forwarder-test-file") }
    filesNew = Array.new(@random.rand(3..5)){ |i| Stud::Temporary.file("logstash-forwarder-test-file", "a+") }

    file_paths = @files.map do |u|
      { :paths => [u.path] }
    end

    file_paths += filesNew.map do |u|
      { :paths => [u.path] }
    end

    filesNew.each do |file|
      File.delete(file)
    end


    json = {
        :network => {
            :servers => ["localhost:#{@server.port}"],
            "ssl ca" => "#{@ssl_cert.path}",
        },
        :files => file_paths
    }

    @config.puts(json.to_json)
    @config.close

    # initialize files with 1-100 lines of text
    @files.each do |file|
      ensure_delete(file)
      initial = @random.rand(1..100)
      initial.times do |count|
        file.puts("initial #{count}")
      end
      file.close
      file.reopen(file.path, "a+")
    end

    # Start LSF
    startup

    filesNew.each do |file|
      File.open(file, "a+")
      ensure_delete(file)
    end

    @files.concat(filesNew)

    expected_events = Array.new
    count = @random.rand(10000..20000)
    count.times do |i|
      selected_file = @files[@random.rand(@files.length)]
      selected_file.puts("#{i}")
      expected_events << {
          "line" => "#{i}",
          "file" => selected_file.path,
          "host" => Socket.gethostname
      }
    end


    @files.each do |f|
      if not f.closed?
        f.close
      end
    end

    # Wait for logstash-forwarder to finish publishing data to us.
    Stud::try(30.times) do
      raise "have #{@actual_events.size}, want #{count}" if @actual_events.size < count
    end

    # events are not guaranteed to arrive in-order, so sort by line value
    @actual_events = @actual_events.sort_by{ |hsh| Integer(hsh["line"]) }

    # Now verify that we have all the data
    insist { @actual_events.size } == count
    @actual_events.each_index do |index|
      ["line", "file", "host"].each do |property|
        insist { @actual_events[index][property] } == expected_events[index][property]
      end
    end
  end #end test

end

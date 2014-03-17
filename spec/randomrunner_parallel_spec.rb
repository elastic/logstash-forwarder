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

    @rand_lock = Mutex.new
    @active_files_lock = Mutex.new

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
  end # before each

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
    @rand_lock.synchronize {
      r = @random.rand(0.0..1.0)
      # puts ("Thread: #{Thread.current.object_id}  -  r: #{r}")

      probabilities.each_pair do |key, value|
        if r < value
          return key
        end

        r -= value
      end
    }
  end

  def ensure_delete(file)
    @active_files_lock.synchronize {
      @active_files << file
    }
  end

  def rotate_file(file)
    path = file.path
    rotated_path = path + "_" +SecureRandom.uuid
    puts "\nRotating file: #{File.basename(path)}"
    File.rename(path, rotated_path)
    file.close
    ensure_delete(file)

    new_file = File.new(path, "a+")
    ensure_delete(new_file)

    sleep_value = 15

    # Add a little noise to help stop thundering herds
    @rand_lock.synchronize {
      sleep_value += @random.rand(0.0..5.0)
    }
    sleep(sleep_value)

    return new_file
  end

  it "should follow multiple explicit files from the end, no rotations, parallel writes" do

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

    expected_events = []
    count = @random.rand(1000..10000)

    threads = []
    @files.each_with_index do |file, index|
      threads << Thread.new {

        events = []
        count.times do |i|
          file.puts("#{(index * count) + i}")
          events << {
              "line" => "#{(index * count) + i}",
              "file" => file.path,
              "host" => Socket.gethostname
          }
        end
        Thread.current[:output] = events
      }
    end

    threads.each do |t|
      t.join
      expected_events.concat(t[:output])
    end

    @files.each do |f|
      f.close
    end

    # Wait for logstash-forwarder to finish publishing data to us.
    Stud::try(30.times) do
      raise "have #{@actual_events.size}, want #{expected_events.size}" if @actual_events.size < expected_events.size
    end

    # events are not guaranteed to arrive in-order, so sort by line value
    @actual_events = @actual_events.sort_by{ |hsh| Integer(hsh["line"]) }

    # Now verify that we have all the data
    insist { @actual_events.size } == expected_events.size
    @actual_events.each_index do |index|
      ["line", "file", "host"].each do |property|
        insist { @actual_events[index][property] } == expected_events[index][property]
      end
    end
  end #end test


  it "should follow multiple files via glob from the end, no rotations, parallel writes" do

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

    expected_events = []
    count = @random.rand(1000..10000)

    threads = []
    @files.each_with_index do |file, index|
      threads << Thread.new {
        events = []
        count.times do |i|
          file.puts("#{(index * count) + i}")
          events << {
              "line" => "#{(index * count) + i}",
              "file" => file.path,
              "host" => Socket.gethostname
          }
        end
        Thread.current[:output] = events
      }
    end

    threads.each do |t|
      t.join
      expected_events.concat(t[:output])
    end

    @files.each do |f|
      f.close
    end

    # Wait for logstash-forwarder to finish publishing data to us.
    Stud::try(30.times) do
      raise "have #{@actual_events.size}, want #{expected_events.size}" if @actual_events.size < expected_events.size
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


  it "should follow multiple explicit files from the end, with rotations, parallel writes" do

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

    expected_events = []
    count = @random.rand(1000..10000)

    threads = []
    @files.each_with_index do |file, index|
      threads << Thread.new {
        events = []
        count.times do |i|
          case generate_event   #is rand() threadsafe?  May need to change this
            when :emit
              file.puts("#{(index * count) + i}")
              events << {
                  "line" => "#{(index * count) + i}",
                  "file" => file.path,
                  "host" => Socket.gethostname
              }
            when :rotate
              file = rotate_file(file)
          end
        end
        file.close
        Thread.current[:output] = events
      }
    end

    threads.each do |t|
      t.join
      expected_events.concat(t[:output])
    end

    # Wait for logstash-forwarder to finish publishing data to us.
    Stud::try(30.times) do
      raise "have #{@actual_events.size}, want #{expected_events.size}" if @actual_events.size < expected_events.size
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


  it "should follow multiple files via glob from the end, with rotations, parallel writes" do

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

    expected_events = []
    count = @random.rand(1000..10000)

    threads = []
    @files.each_with_index do |file, index|
      threads << Thread.new {
        events = []
        count.times do |i|
          case generate_event   #is rand() threadsafe?  May need to change this
            when :emit
              file.puts("#{(index * count) + i}")
              events << {
                  "line" => "#{(index * count) + i}",
                  "file" => file.path,
                  "host" => Socket.gethostname
              }
            when :rotate
              file = rotate_file(file)
          end
        end
        file.close
        Thread.current[:output] = events
      }
    end

    threads.each do |t|
      t.join
      expected_events.concat(t[:output])
    end

    # Wait for logstash-forwarder to finish publishing data to us.
    Stud::try(30.times) do
      raise "have #{@actual_events.size}, want #{expected_events.size}" if @actual_events.size < expected_events.size
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

end

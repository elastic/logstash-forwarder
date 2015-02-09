# encoding: utf-8
#
$: << File.realpath(File.join(File.dirname(__FILE__), "..", "lib"))
require "json"
require "lumberjack/server"
require "stud/try"
require "stud/temporary"

describe "lumberjack" do
  # TODO(sissel): Refactor this to use factory pattern instead of so many 'let' statements.
  let(:ssl_certificate) { Stud::Temporary.pathname("ssl_certificate") }
  let(:ssl_key) { Stud::Temporary.pathname("ssl_key") }
  let(:config_file) { Stud::Temporary.pathname("config_file") }
  let(:input_file) { Stud::Temporary.pathname("input_file") }

  let(:lsf) do
    # Start the process, return the pid
    lsf = IO.popen(["./logstash-forwarder", "-config", config_file, "-quiet"])
  end

  let(:random_field) { (rand(30)+1).times.map { (rand(26) + 97).chr }.join }
  let(:random_value) { (rand(30)+1).times.map { (rand(26) + 97).chr }.join }
  let(:port) { rand(50000) + 1024 }

  let(:server) do 
    Lumberjack::Server.new(:ssl_certificate => ssl_certificate, :ssl_key => ssl_key, :port => port)
  end


  let(:logstash_forwarder_config) do
    <<-CONFIG
    {
      "network": {
        "servers": [ "localhost:#{port}" ],
        "ssl ca": "#{ssl_certificate}"
      },
      "files": [
        {
          "paths": [ "#{input_file}" ],
          "fields": { #{random_field.to_json}: #{random_value.to_json} }
        }
      ]
    }
    CONFIG
  end

  after do
    [ssl_certificate, ssl_key, config_file].each do |path|
      File.unlink(path) if File.exists?(path)
    end
    Process::kill("KILL", lsf.pid)
    Process::wait(lsf.pid)
  end

  before do
    system("openssl req -x509  -batch -nodes -newkey rsa:2048 -keyout #{ssl_key} -out #{ssl_certificate} -subj /CN=localhost > /dev/null 2>&1")
    
    File.write(config_file, logstash_forwarder_config)
    lsf

    # Make sure lsf hasn't crashed
    5.times do
      # Sending signal 0 will throw exception if the process is dead.
      Process.kill(0, lsf.pid)
      sleep(rand * 0.1)
    end
  end # before each

  let(:connection) { server.accept }

  it "should follow a file and emit lines as events" do
    # TODO(sissel): Refactor this once we figure out a good way to do
    # multi-component integration tests and property tests.
    fd = File.new(input_file, "wb")
    lines = [ "Hello world", "Fancy Pants", "Some Unicode Emoji: 👍 💗 " ]
    lines.each { |l| fd.write(l + "\n") }
    fd.flush
    fd.close

    # TODO(sissel): Make sure this doesn't take forever, do a timeout.
    count = 0
    events = []
    connection.run do |event|
      events << event
      connection.close if events.length == lines.length
    end

    expect(events.count).to(eq(lines.length))
    lines.zip(events).each do |line, event|
      # TODO(sissel): Resolve the need for this hack.
      event["line"].force_encoding("UTF-8")
      expect(event["line"]).to(eq(line))
      expect(event[random_field]).to(eq(random_value))
    end
  end
end

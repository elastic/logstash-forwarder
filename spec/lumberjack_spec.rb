# encoding: utf-8
#
$: << File.realpath(File.join(File.dirname(__FILE__), "..", "lib"))
require "json"
require "lumberjack/server"
require "stud/try"
require "stud/temporary"

shared_examples_for "logstash-forwarder" do
  # TODO(sissel): Refactor this to use factory pattern instead of so many 'let' statements.
  let(:workdir) { Stud::Temporary.directory }
  let(:ssl_certificate) { File.join(workdir, "certificate.pem") }
  let(:ssl_key) { File.join(workdir, "certificate.key") }
  let(:config_file) { File.join(workdir, "config.json") }
  let(:input_file) { File.join(workdir, "input.log") }

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
    #Calling this method raises a SystemCallError if there are no child processes.
    Process::wait(lsf.pid) rescue ''
  end

  let(:openssl_config_text) do 
    <<-CONFIG
    [req]
    distinguished_name = req_distinguished_name
    req_extensions = v3_req
     
    [req_distinguished_name]
    # Do NOT change these.
    countryName = Country Name (2 letter code)
    stateOrProvinceName = State or Province Name (full name)
    localityName = Locality Name (eg, city)
    organizationName = Organization
    organizationalUnitName  = Organizational Unit Name (eg, section)
    commonName = Common Name
    emailAddress = Email (optional)
     
    # Instead change these, and those should be the most common + required ones 
    # all but OU and Email = req
    countryName_default = US
    stateOrProvinceName_default = CA
    localityName_default = Change_this
    organizationName_default = Change_this
    organizationalUnitName_default = Change_this
    commonName_default = CN
    commonName_max  = 64
    emailAddress_default = mail_for_ca_requesting_team
     
    [ v3_req ]
    # Extensions to add to a certificate request
    basicConstraints = CA:FALSE
    keyUsage = nonRepudiation, digitalSignature, keyEncipherment
    subjectAltName = @alt_names
     
    [alt_names]
    DNS.1 = localhost
    IP.1 = #{ip}
    CONFIG
  end
  let(:openssl_config) { File.join(workdir, "openssl.conf") }

  before do
    File.write(openssl_config, openssl_config_text)
    system("openssl req -x509  -batch -nodes -newkey rsa:2048 -keyout #{ssl_key} -out #{ssl_certificate} -config #{openssl_config} #{redirect}")
    puts "---"
    puts openssl_config_text
    puts "---"
    expect($?).to(be_success)
    File.write(config_file, logstash_forwarder_config)
    lsf

    # Make sure lsf hasn't crashed
    5.times do
      # Sending signal 0 will throw exception if the process is dead.
      Process.kill(0, lsf.pid)
      sleep(rand * 0.1)
    end
  end # before each


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
    p "Accepting..."
    connection = server.accept
    p "Got it" => connection
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

describe "operation" do
  let(:redirect) { ENV["DEBUG"] ? "" : "> /dev/null 2>&1" }
  context "when compiled from source" do
    let(:lsf) do
      # Start the process, return the pid
      IO.popen(["./logstash-forwarder", "-config", config_file, "-quiet"])
    end
    let(:ip) { "127.0.0.1" }
    it_behaves_like "logstash-forwarder" do
    end
  end

  context "when installed from a deb", :deb => true do
    let (:deb) { Dir.glob(File.join(File.dirname(__FILE__), "..", "*.deb")).first }
    let(:container_name) { "lsf-spec-#{$$}" }
    let(:lsf) do
      args = ["docker", "run", "--name", container_name, "-v", "#{workdir}:#{workdir}", "-i", "ubuntu:14.04", "/bin/bash"]
      IO.popen(args, "wb")
    end

    # Have to try repeatedly here because the network configuration of a docker container isn't available immediately.
    let(:ip) do 
      lsf
      ip = nil
      10.times do
        ip = JSON.parse(`docker inspect #{container_name}`)[0]["NetworkSettings"]["Gateway"] rescue nil
        break unless ip.nil? || ip.empty?
        sleep 0.01
      end
      raise "Something is wrong with docker" if ip.nil?
      p :ip => ip
      ip
    end

    it_behaves_like "logstash-forwarder" do
      before do
        if !File.exist?("logstash-forwarder")
          system("make logstash-forwarder #{redirect}") 
          expect($?).to(be_success)
        end
        system("make deb #{redirect}")
        expect($?).to(be_success)
        expect(File).to(be_exist(deb))
        
        FileUtils.cp(deb, workdir)
        lsf.write("dpkg -i #{workdir}/#{File.basename(deb)}\n")
        system("docker inspect #{container_name}")

        # Put a custom config for testing
        lsf.write("sed -e 's/localhost:/#{ip}:/' #{config_file} > /etc/logstash-forwarder.conf\n")

        # Start lsf
        lsf.write("/etc/init.d/logstash-forwarder start\n")

        # Watch the logs
        lsf.write("tail -F /var/log/logstash-forwarder.{err,log}\n")
      end

      after do
        system("docker", "kill", container_name)
      end
    end
  end
end

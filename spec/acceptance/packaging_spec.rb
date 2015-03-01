# encoding: utf-8
describe "packaging" do
  let(:redirect) { ENV["DEBUG"] ? "" : "> /dev/null 2>&1" }
  let(:version) { `./logstash-forwarder -version`.chomp }
  before do
    if !File.exist?("logstash-forwarder")
      system("make logstash-forwarder")
    end
  end

  describe "make rpm" do
    let(:architecture) { RbConfig::CONFIG["host_cpu"] }
    it "should build an rpm" do
      system("make rpm #{redirect}")
      expect($?).to be_success
      expect(File).to be_exist("logstash-forwarder-#{version}-1.#{architecture}.rpm")
    end
  end

  describe "make deb" do
    let(:architecture) do
      a = RbConfig::CONFIG["host_cpu"]
      case a
        when "x86_64"; "amd64" # why? Because computers.
        else a
      end
    end
    it "should build a deb" do
      system("make deb #{redirect}")
      expect($?).to be_success
      expect(File).to be_exist("logstash-forwarder_#{version}_#{architecture}.deb")
    end
  end
end


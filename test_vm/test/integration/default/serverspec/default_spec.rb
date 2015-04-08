require_relative 'spec_helper'

require 'serverspec'


context 'Verify that logstash-forwarder is RUNNING...' do
  describe process("logstash-forwarder") do
    it { should be_running }
  end
end

context 'Files that should be EXCLUDED...' do
  describe command("lsof -X -c logstash- | egrep -v '\.logstash-forwarder' | egrep -E '[0-9]+r\s+REG' | awk '{print $9}'") do
    its(:stdout) { should_not match %r[/var/log/test1.log] }
    its(:stdout) { should_not match %r[/var/log/test3.log] }
  end
end

context ' files that should be INCLUDED...' do
  describe command("lsof -X -c logstash- | egrep -v '\.logstash-forwarder' | egrep -E '[0-9]+r\s+REG' | awk '{print $9}'") do
    its(:stdout) { should match %r[/var/log/test2.log] }
    its(:stdout) { should match %r[/var/log/test4.log] }
    its(:stdout) { should match %r[/var/log/some-other-file1.log] }
  end
end


# encoding: utf-8
require 'spec_helper'
require 'lumberjack/client'
require 'lumberjack/server'
require "socket"
require "thread"
require "openssl"
require "zlib"

describe "Lumberjack::Client" do

  describe "Lumberjack::Socket" do

    let(:port)   { 5000 }

    subject(:socket) { Lumberjack::Socket.new(:port => port, :ssl_certificate => "" ) }

    before do
      allow_any_instance_of(Lumberjack::Socket).to receive(:connection_start).and_return(true)
    end

    context "sequence" do

     let(:hash)   { {:a => 1, :b => 2}}
     let(:max_unsigned_int) { (2**32)-1 }

      before(:each) do
        allow(socket).to receive(:ack).and_return(true)
        allow(socket).to receive(:write).and_return(true)
      end

      it "force sequence to be an unsigned 32 bits int" do
        socket.instance_variable_set(:@sequence, max_unsigned_int)
        socket.write_hash(hash)
        expect(socket.sequence).to eq(1)
      end
    end

    context "ack" do

      let(:hash)   { {:a => 1, :b => 2}}

      before(:each) do
        allow(socket).to receive(:write).and_return(true)
      end

      it "increments the sequence per windows size" do
        allow(socket).to receive(:read_version_and_type).and_return([1, 'A'])
        expect(socket).to receive(:ack).twice.and_call_original

        [5000, 10000].each do |last_ack|
          windows_size = 5001

          allow(socket).to receive(:read_last_ack).and_return(last_ack)

          windows_size.times do
            socket.write_hash(hash)
          end
        end

      end
    end
  end

  describe Lumberjack::Encoder do
    it 'should creates frames without truncating accentued characters' do
      content = {
        "message" => "Le Canadien de Montréal est la meilleure équipe au monde!",
        "other" => "éléphant"
      }
      parser = Lumberjack::Parser.new
      parser.feed(Lumberjack::Encoder.to_frame(content, 0)) do |code, sequence, data|
        expect(data["message"].force_encoding('UTF-8')).to eq(content["message"])
        expect(data["other"].force_encoding('UTF-8')).to eq(content["other"])
      end
    end

    it 'should creates frames without dropping multibytes characters' do
      content = {
        "message" => "国際ホッケー連盟" # International Hockey Federation
      }
      parser = Lumberjack::Parser.new
      parser.feed(Lumberjack::Encoder.to_frame(content, 0)) do |code, sequence, data|
        expect(data["message"].force_encoding('UTF-8')).to eq(content["message"])
      end
    end

    it 'should creates compressed frames' do
      content = {
        "message" => "国際ホッケー連盟" # International Hockey Federation
      }
      parser = Lumberjack::Parser.new
      parser.feed(Lumberjack::Encoder.to_compressed_frame(content, 0)) do |code, sequence, data|
        expect(data["message"].force_encoding('UTF-8')).to eq(content["message"])
      end
    end
  end
end

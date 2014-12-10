require 'spec_helper'
require 'lumberjack/client'
require 'lumberjack/server'

describe Lumberjack::Encoder do
  it 'should creates frames without truncating accentued characters' do
    content = { 
      "message" => "Le Canadien de Montréal est la meilleur équipe au monde!",
      "other" => "éléphant"
    }
    
    parser = Lumberjack::Parser.new
    parser.feed(Lumberjack::Encoder.to_frame(content, 0)) do |code, sequence, data|
      expect(data["message"].force_encoding('UTF-8')).to eq(content["message"])
      expect(data["other"].force_encoding('UTF-8')).to eq(content["other"])
    end
  end
end

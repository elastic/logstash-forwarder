Gem::Specification.new do |gem|
  gem.authors       = ["Jordan Sissel"]
  gem.email         = ["jls@semicomplete.com"]
  gem.description   = "lumberjack log transport library"
  gem.summary       = gem.description
  gem.homepage      = "https://github.com/jordansissel/lumberjack"

  gem.files = %w{
    lib/lumberjack/server.rb
    lib/lumberjack/client.rb
  }
    #lib/lumberjack/server2.rb

  gem.test_files    = []
  gem.name          = "jls-lumberjack"
  gem.require_paths = ["lib"]
  gem.version       = "0.0.20"

  # This isn't used yet because the new protocol isn't ready
  #gem.add_runtime_dependency "ffi-rzmq", "~> 1.0.0"
end

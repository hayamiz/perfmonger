# -*- mode: ruby -*-

$:.push File.expand_path("../lib", __FILE__)
require 'perfmonger/version'

Gem::Specification.new do |s|
  s.name        = 'perfmonger'
  s.version     = PerfMonger::VERSION
  s.summary     = "yet anothor performance measurement/monitoring tool"
  s.description = "yet anothor performance measurement/monitoring tool"
  s.authors     = ["Yuto HAYAMIZU"]
  s.email       = 'y.hayamizu@gmail.com'
  s.homepage    = 'http://github.com/hayamiz/perfmonger/'
  s.license     = 'GPL-2'

  s.required_ruby_version = '>= 1.8.7'

  if RUBY_VERSION < '1.9'
    s.add_dependency "json"
  end
  s.add_development_dependency "bundler"
  s.add_development_dependency "rake"
  s.add_development_dependency "rspec"
  s.add_development_dependency "rake-compiler"

  s.files       = `git ls-files`.split("\n")
  s.files      += Dir.glob("lib/exec/*")
  s.test_files  = `git ls-files -- {test,spec,features}/*`.split("\n")
  s.executables = `git ls-files -- bin/*`.split("\n").map{|f| File.basename(f)}

  s.post_install_message = <<EOS

============================================================

Thank you for installing perfmonger.
Try to start performance monitoring with:

    perfmonger record

Enjoy.

============================================================

EOS

end

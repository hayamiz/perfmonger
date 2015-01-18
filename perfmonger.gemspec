# -*- mode: ruby -*-

Gem::Specification.new do |s|
  s.name        = 'perfmonger'
  s.version     = '0.6.0'
  s.date        = '2015-01-19'
  s.summary     = "yet anothor performance measurement/monitoring tool"
  s.description = "yet anothor performance measurement/monitoring tool"
  s.authors     = ["Yuto HAYAMIZU"]
  s.email       = 'y.hayamizu@gmail.com'
  s.homepage    = 'http://github.com/hayamiz/'
  s.license     = 'GPL-2'

  s.add_development_dependency "rake"
  s.add_development_dependency "rspec"
  s.add_development_dependency "rake-compiler"

  s.files       = `git ls-files`.split("\n")
  s.test_files  = `git ls-files -- {test,spec,features}/*`.split("\n")
  s.executables = `git ls-files -- bin/*`.split("\n").map{|f| File.basename(f)}

  s.extensions << 'ext/perfmonger/extconf.rb'
end

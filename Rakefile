require 'rubygems'
require 'rspec/core/rake_task'
require "bundler/gem_tasks"

task :default => [:spec, :test_core]

desc "Run all specs in spec directory"
RSpec::Core::RakeTask.new(:spec)

task :spec => [:cross_build_core]

desc "Cross build core recorder/player"
task :cross_build_core do
  puts "Buildling binaries for each platform"
  Dir.chdir("./core") do
    sh "./build.sh"
    sh "make"
  end
end

task :build => :cross_build_core

desc "Install Golang libraries"
task :go_get do
  sh "go get -u github.com/hayamiz/go-projson"
  sh "go get -u github.com/hayamiz/perfmonger/core/subsystem"
  sh "go get -u golang.org/x/crypto/ssh/terminal"
  sh "go get -u github.com/mattn/go-isatty"
  sh "go get -u github.com/nsf/termbox-go"
  sh "go get -u github.com/jroimartin/gocui"
end

desc "Run tests of core recorder/player"
task :test_core => [:cross_build_core] do
  Dir.chdir("./core/subsystem") do
    sh "go test -v -cover"

    # running static analysis
    sh "go vet *.go"
  end

  Dir.chdir("./core") do
    sh "go vet *.go"
  end

end

desc "Removed generated files"
task :clean do
  sh "make -C core clean"
end


require 'rubygems'
require 'rspec/core/rake_task'
require "bundler/gem_tasks"

task :default => [:spec]

desc "Run all specs in spec directory"
RSpec::Core::RakeTask.new(:spec)

desc "Build ext"
task :build_ext do
  Dir.chdir("ext/perfmonger") do
    sh "ruby extconf.rb"
    sh "make"
  end
end

task :spec => [:build_ext]

desc "Build core recorder/player"
task :build_core do
  puts "Buildling binaries for each platform"
  Dir.chdir("./core") do
    sh "./build.sh"
  end
end

task :build => :build_core

desc "Run tests of core recorder/player"
task :test_core do
  Dir.chdir("./core/subsystem") do
    sh "go test"
  end
end

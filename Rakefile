require 'rubygems'
require 'rspec/core/rake_task'
require "bundler/gem_tasks"

task :default => [:spec, :test_core]

desc "Run all specs in spec directory"
RSpec::Core::RakeTask.new(:spec)

task :spec => [:self_build_core]

desc "Cross build core recorder/player"
task :cross_build_core do
  puts "Buildling binaries for each platform"
  Dir.chdir("./core") do
    sh "./build.sh"
  end
end

desc "Self build core recorder/player"
task :self_build_core do
  Dir.chdir("./core") do
    sh "./build.sh -"
  end
end

task :build => :cross_build_core

desc "Run tests of core recorder/player"
task :test_core do
  Dir.chdir("./core/subsystem") do
    sh "go test -v -cover"
  end
end

desc "Removed generated files"
task :clean do
  sh "make -C core clean"
end


require 'rubygems'
require 'rspec/core/rake_task'
require "bundler/gem_tasks"

task :default => [:spec]

desc "Run all specs in spec directory"
RSpec::Core::RakeTask.new(:spec)

desc "Build core recorder/player"
task :build_core do
  puts "Buildling binaries for each platform"
  Dir.chdir("./core") do
    sh "./build.sh"
  end
end

task :build => :build_core


require 'optparse'
require 'json'
require 'webrick'
require 'thread'
require 'tmpdir'
require 'fileutils'
require 'erb'

# monkey patching HTTPResponse for server-sent events
class WEBrick::HTTPResponse
  CRLF = "\r\n"
  def send_body_io(socket)
    begin
      if @request_method == "HEAD"
        # do nothing
      elsif chunked?
        begin
          buf  = ''
          data = ''
          while true
            @body.readpartial( @buffer_size, buf ) # there is no need to clear buf?
            data << format("%x", buf.bytesize) << CRLF
            data << buf << CRLF
            _write_data(socket, data)
            data.clear
            @sent_size += buf.bytesize
          end
        rescue EOFError # do nothing
        rescue IOError # do nothing
        end
        _write_data(socket, "0#{CRLF}#{CRLF}")
      else
        size = @header['content-length'].to_i
        _send_file(socket, @body, 0, size)
        @sent_size = size
      end
    ensure
      begin; @body.close; rescue IOError; end
    end
  end
end


module PerfMonger
module Command

class ServerCommand < BaseCommand
  register_command 'server', 'Launch self-contained HTML5 realtime graph server'

  def initialize
    @parser = OptionParser.new
    @parser.banner = <<EOS
Usage: perfmonger server [options] -- [perfmonger record options]

Launch TCP and HTTP server which enables users to get current
perfmonger data record via network.

Options:
EOS

    @hostname = `hostname -f`.strip
    @http_hostname = nil
    @http_port = 20202
    @tcp_port  = 20203
  end

  def parse_args(argv)
    @parser.on('-H', '--hostname NAME', "Host name to display (default: #{@hostname})") do |hostname|
      @hostname = hostname
    end

    @parser.on('--http-host NAME',
               "Host name for HTTP server URL. If not specified, value of '--hostname' option is used.") do |hostname|
      @http_hostname = hostname
    end

    @parser.on('--port PORT', 'HTTP server port to listen.') do |port|
      if ! port =~ /\A\d+\Z/
        puts("ERROR: invalid port number value: #{port}")
        puts(@parser.help)
        exit(false)
      end
      @http_port = port.to_i
    end

    @parser.on('-h', '--help', 'Show this help.') do
      puts @parser.help
      exit(false)
    end

    # @parser.on('--tcp-port PORT', 'TCP data server port to listen.') do |port|
    #   if ! port =~ /\A\d+\Z/
    #     puts("ERROR: invalid port number value: #{port}")
    #     puts(@parser.help)
    #     exit(false)
    #   end
    #   @tcp_port = port.to_i
    # end

    @parser.parse!(argv)

    if @http_hostname.nil?
      @http_hostname = @hostname
    end

    @record_cmd_args = argv
  end

  def run(argv)
    tmp_rootdir = Dir.mktmpdir
    parse_args(argv)

    _, record_option = PerfMonger::Command::RecordOption.parse(@record_cmd_args)

    # find perfmonger command
    perfmonger_bin = File.expand_path('bin/perfmonger', PerfMonger::ROOTDIR)
    if ! File.executable?(perfmonger_bin)
      puts("ERROR: perfmonger not found!")
      exit(false)
    end

    record_cmd = [perfmonger_bin, 'record',
                  *@record_cmd_args]

    @recorder = Recorder.new(record_cmd).start

    puts("PerfMonger Realtime Monitor: http://#{@http_hostname}:#{@http_port}/dashboard")
    puts("")

    @http_server =  WEBrick::HTTPServer.new({:DocumentRoot => tmp_rootdir,
                                              :BindAddress => '0.0.0.0',
                                              :Port => @http_port})
    setup_webrick(@http_server, record_option)

    trap(:INT) do
      @http_server.stop
      @recorder.stop
    end
    @http_server.start
  ensure
    FileUtils.rm_rf(tmp_rootdir)
  end

  private
  class Recorder
    def initialize(record_cmd)
      @current_record = nil
      @mutex = Mutex.new
      @cond = ConditionVariable.new
      @thread = nil
      @record_cmd = record_cmd

      @working = false
    end

    def start
      @mutex.synchronize do
        if @thread.nil?
          @thread = true
          @working = true
        else
          return
        end
      end

      @thread = Thread.start do
        begin
          IO.popen(@record_cmd, "r") do |io|
            io.each_line do |line|
              @mutex.synchronize do
                @current_perf_data = line.strip
                @cond.broadcast
              end
            end
          end
        rescue Exception => err
          puts("ERROR: Exception in record_thread(#{@record_thread}) in perfmonger-server")
          puts("#{err.class.to_s}: #{err.message}")
          puts(err.backtrace)
        end
      end

      self
    end

    def stop
      @mutex.synchronize do
        @working = false
        @thread.terminate
        @cond.broadcast
      end
    end

    def get_current_record
      @mutex.synchronize do
        if @working
          @cond.wait(@mutex)
          current_perf_data = @current_perf_data
        else
          raise EOFError
        end
      end
    end
  end

  class DashboardServlet < WEBrick::HTTPServlet::AbstractServlet
    def initialize(server, assets_dir, record_option, opt = {})
      @assets_dir = assets_dir
      @record_option = record_option
      @opt = opt
      super
    end

    def escape_device_name(dev)
      dev.gsub(' ', '_').gsub('-', '_')
    end

    def do_GET(req, res)
      res.content_type = 'text/html'
      res['cache-control'] = 'no-cache'

      # Variables for erb template
      devices = @record_option.devices
      report_cpu = @record_option.report_cpu
      hostname = @opt[:hostname]

      erb = ERB.new(File.read(File.expand_path('dashboard.erb', @assets_dir)))
      res.body = erb.result(Kernel.binding)
    end
  end

  class FaucetServlet < WEBrick::HTTPServlet::AbstractServlet
    def initialize(server, recorder)
      super(server)
      @recorder = recorder
    end

    def do_GET(req, res)
      res.chunked = true
      res.content_type = 'text/event-stream'
      res['cache-control'] = 'no-cache'
      r, w = IO.pipe

      Thread.start do
        begin
          while record = @recorder.get_current_record
            w << "data: " << record << "\r\n" << "\r\n"
          end
        rescue Errno::EPIPE
          # puts("Connection closed for /faucet")
          # connection closed
        rescue EOFError
          # puts("Recorder has been terminated")
          # connection closed
        rescue Exception => err
          puts("ERROR: Exception in faucet pipe writer")
          puts("#{err.class.to_s}: #{err.message}")
          puts(err.backtrace)
        ensure
          # puts("[FaucetServlet][do_GET] close w,r pipe")
          begin; w.close; rescue IOError; end
          begin; r.close; rescue IOError; end
        end
      end

      res.body = r
    end
  end

  def setup_webrick(webrick_server, record_option)
    # find assets dir
    # Search build environment first, then installed dir
    assets_dir = [File.expand_path('../../../../../data/assets', __FILE__),
                  File.expand_path('assets', PerfMonger::DATAROOTDIR)].find do |dir|
      File.directory?(dir)
    end
    if assets_dir.nil?
      puts("ERROR: Assets for PerfMonger monitor not found!")
      exit(false)
    end

    webrick_server.mount_proc('/') do |req, res|
      puts("Request Path: " + req.path)
      if req.path == '/favicon.ico'
        res.set_redirect(WEBrick::HTTPStatus::TemporaryRedirect, '/assets/favicon.ico')
      else
        res.set_redirect(WEBrick::HTTPStatus::TemporaryRedirect, '/dashboard')
      end
    end
    webrick_server.mount('/dashboard', DashboardServlet, assets_dir, record_option,
                         :hostname => @hostname)
    webrick_server.mount('/assets', WEBrick::HTTPServlet::FileHandler, assets_dir)
    webrick_server.mount('/faucet', FaucetServlet, @recorder)
  end
end

end # module Command
end # module PerfMonger

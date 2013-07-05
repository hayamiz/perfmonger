
require 'optparse'
require 'json'
require 'webrick'
require 'thread'
require 'tmpdir'
require 'fileutils'
require 'base64'

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
        end
        _write_data(socket, "0#{CRLF}#{CRLF}")
      else
        size = @header['content-length'].to_i
        _send_file(socket, @body, 0, size)
        @sent_size = size
      end
    ensure
      @body.close
    end
  end
end


module PerfMonger
module Command

class ServerCommand < BaseCommand
  register_command 'server'

  def initialize
    @parser = OptionParser.new
    @parser.banner = <<EOS
Usage: perfmonger server [options] -- [perfmonger record options]

Launch TCP and HTTP server which enables users to get current
perfmonger data record via network.

Options:
EOS

    @http_port = 20202
    @tcp_port  = 20203
  end

  def parse_args(argv)
    @parser.on('--http-port PORT') do |port|
      if ! port =~ /\A\d+\Z/
        puts("ERROR: invalid port number value: #{port}")
        puts(@parser.help)
        exit(false)
      end
      @http_port = port.to_i
    end

    @parser.on('--tcp-port PORT') do |port|
      if ! port =~ /\A\d+\Z/
        puts("ERROR: invalid port number value: #{port}")
        puts(@parser.help)
        exit(false)
      end
      @tcp_port = port.to_i
    end

    @parser.parse!(argv)

    @record_cmd_args = argv
  end

  def run(argv)
    tmp_rootdir = Dir.mktmpdir
    parse_args(argv)
    load_frontend_files()

    # find perfmonger command
    perfmonger_bin = File.expand_path('../../../../perfmonger', __FILE__)
    if ! File.executable?(perfmonger_bin)
      perfmonger_bin = File.expand_path('perfmonger', PerfMonger::BINDIR)
    end
    if ! File.executable?(perfmonger_bin)
      puts("ERROR: perfmonger(1) not found!")
      exit(false)
    end

    record_cmd = [perfmonger_bin, 'record',
                  *@record_cmd_args]

    @recorder = Recorder.new(record_cmd).start

    @http_server =  WEBrick::HTTPServer.new({:DocumentRoot => tmp_rootdir,
                                              :BindAddress => '0.0.0.0',
                                              :Port => @http_port})
    setup_webrick(@http_server)

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

    def current_record
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
    def do_GET(req, res)
      res.content_type = 'text/html'
      res.body = $frontend_files['/html/dashboard']
      res['cache-control'] = 'no-cache'
    end
  end

  class AssetsServlet < WEBrick::HTTPServlet::AbstractServlet
    def do_GET(req, res)
      if ! $frontend_files[req.path]
        puts "Not found: #{req.path}"
        res.status = 404
        res.content_type = 'text/plain'
        res.body = "404 Not Found: #{req.path}"
        return
      end

      res.content_type = case req.path
                         when /\.css\Z/
                           "text/css"
                         when /\.js\Z/
                           "text/javascript"
                         when /\.png\Z/
                           "image/png"
                         else
                           "text/plain"
                         end

      res.body = $frontend_files[req.path]
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
          while record = @recorder.current_record
            w << "data: " << record << "\r\n" << "\r\n"
          end
        rescue Errno::EPIPE
          puts("Connection closed for /faucet")
          # connection closed
        rescue EOFError
          puts("Recorder has been terminated")
          # connection closed
        rescue Exception => err
          puts("ERROR: Exception in faucet pipe writer")
          puts("#{err.class.to_s}: #{err.message}")
          puts(err.backtrace)
        end
      end

      res.body = r
    end
  end

  def setup_webrick(webrick_server)
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
        res.set_redirect(WEBrick::HTTPStatus::TemporaryRedirect, '/html/dashboard')
      end
    end
    webrick_server.mount('/html/dashboard', DashboardServlet)
    webrick_server.mount('/assets', WEBrick::HTTPServlet::FileHandler, assets_dir)
    webrick_server.mount('/faucet', FaucetServlet, @recorder)
  end

  def load_frontend_files()
$frontend_files = Hash.new
def register_file(path, content)
  $frontend_files[path] = content
end

register_file '/html/dashboard', <<EOS
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>Bootstrap, from Twitter</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="description" content="">
    <meta name="author" content="">

    <!-- Le styles -->
    <link href="../assets/css/bootstrap.css" rel="stylesheet">
    <style>
      body {
        padding-top: 60px; /* 60px to make the container go all the way to the bottom of the topbar */
      }
    </style>

    <!-- HTML5 shim, for IE6-8 support of HTML5 elements -->
    <!--[if lt IE 9]>
      <script src="../assets/js/html5shiv.js"></script>
    <![endif]-->

    <!-- Fav and touch icons -->
    <link rel="apple-touch-icon-precomposed" sizes="144x144" href="../assets/ico/apple-touch-icon-144-precomposed.png">
    <link rel="apple-touch-icon-precomposed" sizes="114x114" href="../assets/ico/apple-touch-icon-114-precomposed.png">
    <link rel="apple-touch-icon-precomposed" sizes="72x72" href="../assets/ico/apple-touch-icon-72-precomposed.png">
    <link rel="apple-touch-icon-precomposed" href="../assets/ico/apple-touch-icon-57-precomposed.png">
    <link rel="shortcut icon" href="../assets/ico/favicon.png">
  </head>

  <body>

    <div class="navbar navbar-inverse navbar-fixed-top">
      <div class="navbar-inner">
        <div class="container">
          <a class="brand" href="#">PerfMonger Monitor</a>
<!--
          <div class="nav-collapse collapse">
            <ul class="nav">
              <li class="active"><a href="#">Home</a></li>
              <li><a href="#about">About</a></li>
              <li><a href="#contact">Contact</a></li>
            </ul>
          </div><!--/.nav-collapse -->
-->
        </div>
      </div>
    </div>

    <div class="container">
      <div id="graph" style="width: 100%; height: 600px;"></div>
    </div> <!-- /container -->

    <!-- Le javascript
    ================================================== -->
    <!-- Placed at the end of the document so the pages load faster -->
    <script type="text/javascript" src="http://ajax.googleapis.com/ajax/libs/jquery/1.7.2/jquery.min.js"></script>
    <script src="../assets/js/bootstrap.js"></script>
    <script src="../assets/js/canvasjs.min.js"></script>

<pre id="msg"></pre>

<script>
var records = [];
var datapoints = [{x: new Date(new Date() - 1000), y: 0.0}];

var chart = new CanvasJS.Chart("graph",
{
  title:{
    text: "IOPS",
    },
    axisX: {
      valueFormatString: "HH:mm:ss",
      interval:5,
      intervalType: "second",
    },
      axisY:{
        includeZero: true
      },
      data: [
      {
        type: "line",
        dataPoints: datapoints
      },
      ]
    });
chart.render();

function add_record(record) {
  records.push(record);
  datapoints.push({x: new Date(record['time'] * 1000), y: record['ioinfo']['sda']['r/s']});

  last_record = records[records.length - 1];

  while (records[0]['time'] < last_record['time'] - 30.0) {
    records.shift();
    datapoints.shift();
  }

  chart.data = [{
    type: "line",
    dataPoints: records.map(function(r){ return {x: r['time'], y: r['ioinfo']['sda']['r/s']};})
  }];
  chart.render();
}

function draw_message() {
  var e = document.getElementById( "msg" );
  e.textContent = records.map(function(record){ return record['ioinfo']['sda']['r/s'].toString() }).join(", ")
}

var handleMessage = function handleMessage( evt ) {
  var record = JSON.parse(evt.data);
  if (record['ioinfo'] != null) {
    add_record(record);
  }
  // draw_message();
};
var handleEnd = function handleEnd( evt ) {
  evt.currentTarget.close();
}

var source = new EventSource( '/faucet' );
source.addEventListener( 'message', handleMessage, false );
source.addEventListener( 'end'    , handleEnd    , false );
</script>
  </body>
</html>
EOS

 end
end

end # module Command
end # module PerfMonger

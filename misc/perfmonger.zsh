
_perfmonger-record() {
    local curcontext=$curcontext state line ret=1
    declare -A opt_args

    # TODO: match -d,--device multiple times
    _arguments -w -C -S \
        '--background[run in background mode]' \
        '--status[show running perfmonger recording session]' \
        '--kill[kill running perfmonger recording session]' \
        {-d,--device}'[device name to be monitored]: :__perfmonger_devices' \
        '(-i --interval)'{-i,--interval}'[measurement interval]: interval in sec' \
        {-l,--logfile}'[log file name]: :_files' \
        {-B,--no-interval-backoff}'[prevent interval backoff]' \
        {-s,--start-delay}'[wait specified time before starting measurement]: wait time in sec' \
        {-t,--timeout}'[length of measurement time]: timeout in sec' \
        '--no-cpu[Do not monitor CPU usage]' \
        {-v,--verbose}'[verbose]' \
        {-h,--help}'[show help]' \
        && return
}

_perfmonger-stat() {
    local ret
    _call_function ret _perfmonger-record
    return ret
}

_perfmonger-play() {
    local curcontext=$curcontext state line ret=1
    declare -A opt_args

    _arguments -w -C -S \
        '*:: :_files' \
        {-h,--help}'[show help]' \
        && return
}

_perfmonger-summary() {
    local curcontext=$curcontext state line ret=1
    declare -A opt_args

    # TODO: accept only 1 file
    _arguments -w -C -S \
        '--json[output summary in JSON format]' \
        '--disk-only[select disk devices that matches regexp]: regular expression for target devices' \
        {-p,--pager}'[use pager to see summary output]: pager program' \
        {-h,--help}'[show help]' \
        '*:: :_files' \
        && return
}

_perfmonger-plot() {
    local curcontext=$curcontext state line ret=1
    declare -A opt_args

    # TODO: accept only 1 file
    _arguments -w -C -S \
        '--offset-time: offset time in sec' \
        {-o,--output-dir}'[output directory]:_directories' \
        {-T,--output-type}'[output image type]:output type:(pdf png)' \
        {-p,--prefix}'[output file name prefix]:prefix' \
        {-s,--save}'[save gnuplot script and data files]' \
        '--disk-only[select disk devices that matches regexp]: regular expression for target devices' \
        '--disk-read-only[plot only read performance for disks]' \
        '--disk-write-only[plot only write performance for disks]' \
        '--disk-read-write[plot read and write performance for disks]' \
        {-h,--help}'[show help]' \
        '*:: :_files' \
        && return
}

_perfmonger-fingerprint() {
    local curcontext=$curcontext state line ret=1
    declare -A opt_args

    _arguments -w -C -S \
        '*::output tarball:_files -g "*.((tar|TAR)(.gz|.GZ|.Z|.bz2|.lzma|.xz|)|(tbz|tgz|txz))(-.)"' \
        {-h,--help}'[show help]' \
        && return
}

_perfmonger-server() {
    local curcontext=$curcontext state line ret=1
    declare -A opt_args

    # TODO: complete record options after '--'
    _arguments -w -C \
        {-H,--hostname}'[host name to display]:host name' \
        '--http-host[host name for HTTP URL]:host name' \
        '--port[port number]:port number' \
        {-h,--help}'[show help]' \
        && return

    return $ret
}

__perfmonger_devices() {
    _values 'device' $(tail -n +3 /proc/partitions | awk '{print $4}')
}

# main completion function
# (( $+functions[_perfmonger] )) ||
_perfmonger() {
    local curcontext context state line
    declare -A opt_args

    integer ret=1

    _arguments -C -S \
        '(- :)'{-h,--help}'[show help]' \
        '(- :)'{-v,--version}'[show version]' \
        '(-): :->commands' \
        '(-)*:: :->option-or-argument' && return

    case $state in
        (commands)
            _perfmonger_commands && ret=0
            ;;
        (option-or-argument)
            if (( $+functions[_perfmonger-$words[1]] )); then
                _call_function ret _perfmonger-$words[1]
            else
                _message 'unknown sub-command'
            fi
            ;;
    esac

    return $ret
}

_perfmonger_commands() {
    _values 'command' \
        'record[record system performance into a log file]' \
        'play[play a recorded performance log in JSON]' \
        'live[iostat and mpstat equivalent speaking JSON]' \
        'stat[run a command and record system performance during execution]' \
        'plot[plot system performance graphs from a perfmonger log file]' \
        'fingerprint[gather all possible system config information]' \
        'server[launch self-contained HTML5 realtime graph server]' \
        'summary[show summary of a perfmonger log file]'
}

compdef _perfmonger perfmonger

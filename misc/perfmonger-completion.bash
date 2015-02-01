# bash completion support for PerfMonger

_perfmonger() {
    local cmd cur prev subcmd
    cmd=$1
    cur=$2
    prev=$3

    subcmds="record stat plot fingerprint server summary"

    case $prev in
        perfmonger)
            COMPREPLY=( $(compgen -W "$subcmds" $cur) )
            return 0
            ;;
        -d|--device)
            COMPREPLY=( $(tail -n +3 /proc/partitions | awk '{print $4}') )
            return 0
            ;;
        -l|--logfile)
            COMPREPLY=( $(compgen -o default) )
            return 0
            ;;
    esac

    # complete options
    subcmd=${COMP_WORDS[1]}

    case "$cur" in
        # complete options
        -*)
            COMPREPLY=( $(compgen -W "$(perfmonger $subcmd -h | egrep -o ' ([-][[:alnum:]]|-{2}[[:alnum:]-]+)\b')" -- "$cur") )
            return 0
            ;;
        *)
            COMPREPLY=( $(compgen -o default "$cur") )
            return 0
            ;;
    esac

    return 0
}

complete -F _perfmonger perfmonger

_warp_hosts() {
    local cur=${COMP_WORDS[COMP_CWORD]}
    local hosts=$(warp list 2>/dev/null | tail -n +3 | awk '{print $1}')
    COMPREPLY=( $(compgen -W "$hosts" -- "$cur") )
}

_warp() {
    local commands="connect c list ls add a edit ed remove rm export e shell-config completion"
    local cur=${COMP_WORDS[COMP_CWORD]}
    COMPREPLY=( $(compgen -W "$commands" -- "$cur") )
}

complete -F _warp warp
complete -F _warp_hosts c
complete -F _warp_hosts connect

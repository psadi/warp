#compdef warp

_warp_hosts() {
    local -a hosts
    hosts=($(warp list 2>/dev/null | tail -n +3 | awk '{print $1}'))
    _describe 'hosts' hosts
}

_warp_commands() {
    local -a commands
    commands=(
        'connect:Connect to a host'
        'c:Connect to a host'
        'list:List all hosts'
        'ls:List all hosts'
        'add:Add a new host'
        'a:Add a new host'
        'edit:Edit a host'
        'ed:Edit a host'
        'remove:Remove hosts'
        'rm:Remove hosts'
        'export:Export to CSV'
        'e:Export to CSV'
        'shell-config:Show shell integration'
        'completion:Show shell integration'
    )
    _describe 'commands' commands
}

_warp() {
    local curcontext="$curcontext" state line
    typeset -A opt_args

    _arguments -C \
        '-h[Show help]' \
        '--help[Show help]' \
        '*::command:_warp_commands'
}

_warp

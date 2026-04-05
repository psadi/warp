complete -c warp -f -a "connect c list ls add a edit ed remove rm export e shell-config completion"

function __warp_hosts
    warp list 2>/dev/null | tail -n +3 | awk '{print $1}' | while read -l host
        echo $host
    end
end

complete -c warp -f -n "__fish_seen_subcommand_from connect" -a "(__warp_hosts)"
complete -c warp -f -n "__fish_seen_subcommand_from c" -a "(__warp_hosts)"

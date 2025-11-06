# fh - Fast History
# Bash shell integration
# This file is sourced by ~/.bashrc

# fh save hook - captures command after execution
__fh_save() {
    local exit_code=$?
    local last_cmd=$(HISTTIMEFORMAT='' history 1 | sed 's/^[ ]*[0-9]*[ ]*//')

    # Skip empty commands
    if [[ -z "$last_cmd" ]]; then
        return $exit_code
    fi

    # Skip if this is the same command as last time (prevents duplicates)
    # This handles both: Ctrl-R without execution, and pressing Enter on empty lines
    if [[ "$last_cmd" == "${__fh_last_cmd:-}" ]]; then
        return $exit_code
    fi
    __fh_last_cmd="$last_cmd"

    # Save to fh in background to avoid blocking the prompt
    {
        fh --save \
            --cmd "$last_cmd" \
            --exit-code $exit_code \
            --duration 0 \
            2>/dev/null
    } &
    disown

    return $exit_code
}

# Add to PROMPT_COMMAND if not already present
if [[ "$PROMPT_COMMAND" != *"__fh_save"* ]]; then
    if [[ -z "$PROMPT_COMMAND" ]]; then
        PROMPT_COMMAND="__fh_save"
    else
        PROMPT_COMMAND="__fh_save; $PROMPT_COMMAND"
    fi
fi

# Bind Ctrl-R to fh
# Note: Requires bash 4.0+ for READLINE_LINE to work properly
__fh_widget() {
    local selected
    selected=$(fh < /dev/tty)
    READLINE_LINE="${selected}"
    READLINE_POINT=${#READLINE_LINE}
}

bind -x '"\C-r": __fh_widget'

# fh - Fast History
# Zsh shell integration
# This file is sourced by ~/.zshrc

# fh save hook - captures command after execution
__fh_save() {
    local exit_code=$?
    local last_cmd=$(fc -ln -1)

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

# Add to precmd_functions if not already present
if (( ! ${precmd_functions[(I)__fh_save]} )); then
    precmd_functions+=(__fh_save)
fi

# fh widget for Ctrl-R
__fh_widget() {
    local selected
    selected=$(fh)
    if [[ -n "$selected" ]]; then
        LBUFFER="$selected"
        zle reset-prompt
    fi
}

# Register the widget
zle -N __fh_widget

# Bind Ctrl-R to fh widget
bindkey '^R' __fh_widget

# fh - Known Bugs

## Active Bugs

(none)

---

## Fixed Bugs

### Bug #3: Ctrl-R selected commands saved without execution
**Status**: Fixed
**Priority**: High
**Reported**: 2025-11-05
**Affects**: bash shell integration (Ctrl-R widget)

**Symptom**:
When using Ctrl-R to search history and selecting a command but NOT executing it (pressing Escape or Ctrl-C), the selected command still gets saved to fh history. This can result in duplicate entries if the same command is selected multiple times without execution.

**Root Cause**:
1. `__fh_widget()` populates `READLINE_LINE` with the selected command
2. Bash automatically adds this to its own history when the widget returns
3. `PROMPT_COMMAND` runs `__fh_save()` which reads from bash history
4. The command gets saved to fh even though it was never executed

**Impact**:
- Pollutes history with commands that were never run
- Creates duplicate entries
- Confusing for users reviewing their history

**Solution Implemented**:
Track the last saved command (`__fh_last_cmd`) and skip saving if it's identical to the current command. This simple approach handles multiple scenarios:
- Ctrl-R + Ctrl-C (command selected but not executed)
- Pressing Enter on empty prompts (bash keeps last command in history)
- Prevents any duplicate consecutive saves

```bash
# In __fh_save():
if [[ "$last_cmd" == "${__fh_last_cmd:-}" ]]; then
    return $exit_code
fi
__fh_last_cmd="$last_cmd"
```

**Files Modified**:
- `pkg/capture/shell/bash.sh` - Added deduplication logic
- `pkg/capture/shell/zsh.sh` - Added deduplication logic, fixed `fc -ln -1` usage

**Fixed in**: Bug fix session (2025-11-05)

---

## Fixed Bugs

### Bug #2: Ctrl-R selection doesn't populate command line (bash)
**Status**: Fixed
**Priority**: High
**Reported**: 2025-11-05
**Affects**: bash shell integration (Ctrl-R binding)

**Symptom**:
When pressing Ctrl-R, FZF launches correctly and shows history. However, after selecting a command and pressing Enter, the command is NOT populated on the command line. Expected behavior: the selected command should appear on the command line (without executing), just like native Ctrl-R.

**Root Cause**:
The `bind -x` command in bash has limitations with modifying the readline buffer (`READLINE_LINE`). While the code attempts to set `READLINE_LINE` and `READLINE_POINT`, `bind -x` doesn't always properly update the readline state, especially in newer bash versions.

Current implementation:
```bash
__fh_widget() {
    local selected
    selected=$(fh)
    if [[ -n "$selected" ]]; then
        READLINE_LINE="$selected"
        READLINE_POINT=${#READLINE_LINE}
    fi
}

bind -x '"\C-r": __fh_widget'
```

**Solution Options**:

1. **Use `bind -x` with `READLINE_LINE` manipulation** (current approach, not working)

2. **Use a custom readline function** (more complex, better support):
   ```bash
   __fh_readline_widget() {
       local selected
       selected=$(fh < /dev/tty)
       READLINE_LINE="$selected"
       READLINE_POINT=${#READLINE_LINE}
   }

   bind -x '"\C-r": __fh_readline_widget'
   ```

3. **Use readline's `macro` binding** (cleanest):
   ```bash
   __fh_get_command() {
       fh < /dev/tty
   }

   bind '"\C-r": "\C-e\C-u`__fh_get_command`\e\C-e\er\e^"'
   ```
   This approach uses command substitution within a readline macro.

4. **Use a temporary file approach**:
   ```bash
   __fh_widget() {
       local tmpfile=$(mktemp)
       fh > "$tmpfile" < /dev/tty
       READLINE_LINE=$(cat "$tmpfile")
       READLINE_POINT=${#READLINE_LINE}
       rm "$tmpfile"
   }
   ```

**References**:
- Similar issue in fzf: https://github.com/junegunn/fzf/issues/477
- Bash readline documentation on `bind -x` limitations

**Root Cause Analysis**:
Bash 3.2 (macOS default) has a bug where `bind -x` doesn't properly update `READLINE_LINE`. This was fixed in bash 4.0+.

**Fix Applied**:
1. Added `< /dev/tty` to ensure FZF gets proper terminal input
2. Updated README to require bash 4.0+ (bash 3.2 not supported)
3. Added installation instructions for upgrading bash on macOS

```bash
__fh_widget() {
    local selected
    selected=$(fh < /dev/tty)
    READLINE_LINE="${selected}"
    READLINE_POINT=${#READLINE_LINE}
}

bind -x '"\C-r": __fh_widget'
```

**Files Modified**:
- `pkg/capture/shell/bash.sh` - Added `< /dev/tty` and note about bash 4.0+ requirement
- `README.md` - Added bash version requirement and macOS installation instructions

**Requirements**:
- Bash 4.0 or later (macOS users must upgrade via homebrew)

**Fixed in**: Phase 3.1/3.2

---

## Fixed Bugs (Below)

### Bug #1: Background job notifications polluting prompt
**Status**: Fixed
**Priority**: High
**Reported**: 2025-11-05
**Affects**: bash shell integration

**Symptom**:
```bash
$
[4]   Done                    ( fh --save --cmd "$last_cmd" --exit-code $exit_code --duration 0 2> /dev/null )
$
[5]   Done                    ( fh --save --cmd "$last_cmd" --exit-code $exit_code --duration 0 2> /dev/null )
```

**Root Cause**:
The `__fh_save()` function in `pkg/capture/shell/bash.sh` runs `fh --save` in a background subshell:
```bash
(
    fh --save \
        --cmd "$last_cmd" \
        --exit-code $exit_code \
        --duration 0 \
        2>/dev/null
) &
```

Bash prints job control notifications when background jobs complete, polluting the prompt.

**Solution**:
Use `disown` or disable job control notifications for the background process:
```bash
{
    fh --save \
        --cmd "$last_cmd" \
        --exit-code $exit_code \
        --duration 0 \
        2>/dev/null
} &
disown
```

Or use `set +m` to disable monitor mode locally:
```bash
(
    set +m  # Disable job control
    fh --save ... 2>/dev/null &
)
```

**Fix Applied**:
Changed from subshell `( ... ) &` to brace group `{ ... } & disown`:
```bash
{
    fh --save \
        --cmd "$last_cmd" \
        --exit-code $exit_code \
        --duration 0 \
        2>/dev/null
} &
disown
```

The `disown` command removes the background job from the shell's job table, preventing job control notifications.

**Files Modified**:
- `pkg/capture/shell/bash.sh`
- `pkg/capture/shell/zsh.sh`

**Fixed in**: Phase 3.1/3.2

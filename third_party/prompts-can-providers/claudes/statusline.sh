#!/bin/bash
# Claude Code statusline: Dir | Model | Context % | 5h: % (reset) | 7d: % (reset)
exec 2>/dev/null

input=$(cat)

# Parse each field separately to avoid TSV alignment issues
used_tokens=$(printf '%s' "$input" | jq -r '(
  (.context_window.current_usage.input_tokens // 0)
  + (.context_window.current_usage.cache_creation_input_tokens // 0)
  + (.context_window.current_usage.cache_read_input_tokens // 0)
)')
window_size=$(printf '%s' "$input" | jq -r '.context_window.context_window_size // 0')
five_pct=$(printf '%s' "$input" | jq -r '.rate_limits.five_hour.used_percentage // null | if . then (. | round | tostring) else empty end')
five_reset=$(printf '%s' "$input" | jq -r '.rate_limits.five_hour.resets_at // empty | tostring')
seven_pct=$(printf '%s' "$input" | jq -r '.rate_limits.seven_day.used_percentage // null | if . then (. | round | tostring) else empty end')
seven_reset=$(printf '%s' "$input" | jq -r '.rate_limits.seven_day.resets_at // empty | tostring')
model_name=$(printf '%s' "$input" | jq -r '.model.display_name // empty')
dir_name=$(printf '%s' "$input" | jq -r '.workspace.current_dir // empty | split("/")[-1]')
api_duration_ms=$(printf '%s' "$input" | jq -r '.cost.total_api_duration_ms // 0')
output_tokens=$(printf '%s' "$input" | jq -r '.context_window.current_usage.output_tokens // empty')

RESET="\033[0m"
DIM="\033[2m"
GREEN="\033[32m"
YELLOW="\033[33m"
RED="\033[31m"
BLUE="\033[94m"
MAGENTA="\033[95m"
CYAN="\033[36m"

# Format seconds remaining as "4h 23m" or "1d 21h"
format_reset() {
  local ts="$1"
  [ -z "$ts" ] && return
  local epoch now diff
  epoch=$(printf '%s' "$ts" | tr -dc '0-9')
  [ -z "$epoch" ] && return
  now=$(date +%s)
  diff=$((epoch - now))
  [ "$diff" -le 0 ] && return
  local mins=$(( diff / 60 ))
  local hours=$(( mins / 60 ))
  local days=$(( hours / 24 ))
  if [ "$days" -ge 1 ]; then
    printf "%dd%dh" "$days" $(( hours % 24 ))
  elif [ "$hours" -ge 1 ]; then
    printf "%dh%dm" "$hours" $(( mins % 60 ))
  else
    printf "%dm" "$mins"
  fi
}

# Dir part
dir_part=""
if [ -n "$dir_name" ]; then
  dir_part="${CYAN}${dir_name}${RESET}"
fi

# Model part
model_part=""
if [ -n "$model_name" ]; then
  model_part="${DIM}${model_name}${RESET}"
fi

# Context %
ctx_pct=0
if [ "$window_size" -gt 0 ] 2>/dev/null; then
  ctx_pct=$(awk -v u="$used_tokens" -v t="$window_size" 'BEGIN { printf "%d", (u/t)*100 }')
fi
if [ "$ctx_pct" -ge 85 ] 2>/dev/null; then
  ctx_color="$RED"
elif [ "$ctx_pct" -ge 70 ] 2>/dev/null; then
  ctx_color="$YELLOW"
else
  ctx_color="$GREEN"
fi
context_part="${DIM}Ctx${RESET} ${ctx_color}${ctx_pct}%${RESET}"

# Speed part - output tok/s from last request, cached between renders
speed_part=""
cache_file="/tmp/.claude-statusline-speed"
now_ms=$(($(date +%s) * 1000))
if [ -n "$output_tokens" ] && [ "$output_tokens" -gt 0 ] 2>/dev/null; then
  if [ -f "$cache_file" ]; then
    IFS=$'\t' read -r prev_out prev_time prev_speed < "$cache_file"
    delta_tokens=$((output_tokens - prev_out))
    delta_ms=$((now_ms - prev_time))
    if [ "$delta_tokens" -gt 0 ] 2>/dev/null && [ "$delta_ms" -gt 0 ] 2>/dev/null; then
      prev_speed=$(awk -v dt="$delta_tokens" -v dm="$delta_ms" 'BEGIN { printf "%.1f", dt / (dm / 1000) }')
    fi
    if [ -n "$prev_speed" ]; then
      speed_part="${DIM}Out${RESET} ${prev_speed}${DIM} tok/s${RESET}"
    fi
  fi
  printf '%s\t%s\t%s' "$output_tokens" "$now_ms" "${prev_speed:-}" > "$cache_file"
fi

# Usage color
usage_color() {
  local pct="$1"
  if [ "$pct" -ge 90 ] 2>/dev/null; then printf "%s" "$RED"
  elif [ "$pct" -ge 70 ] 2>/dev/null; then printf "%s" "$MAGENTA"
  else printf "%s" "$BLUE"
  fi
}

# 5h part - only show if data available
five_part=""
if [ -n "$five_pct" ]; then
  color=$(usage_color "$five_pct")
  reset_str=$(format_reset "$five_reset")
  if [ -n "$reset_str" ]; then
    five_part="${DIM}5h:${RESET} ${color}${five_pct}%${RESET} ${DIM}(${reset_str})${RESET}"
  else
    five_part="${DIM}5h:${RESET} ${color}${five_pct}%${RESET}"
  fi
fi

# 7d part - only show if data available
seven_part=""
if [ -n "$seven_pct" ]; then
  color=$(usage_color "$seven_pct")
  reset_str=$(format_reset "$seven_reset")
  if [ -n "$reset_str" ]; then
    seven_part="${DIM}7d:${RESET} ${color}${seven_pct}%${RESET} ${DIM}(${reset_str})${RESET}"
  else
    seven_part="${DIM}7d:${RESET} ${color}${seven_pct}%${RESET}"
  fi
fi

# Build output - skip empty parts
parts=()
[ -n "$dir_part" ] && parts+=("$dir_part")
[ -n "$model_part" ] && parts+=("$model_part")
parts+=("$context_part")
[ -n "$speed_part" ] && parts+=("$speed_part")
[ -n "$five_part" ] && parts+=("$five_part")
[ -n "$seven_part" ] && parts+=("$seven_part")

# Join with " | "
output=""
for i in "${!parts[@]}"; do
  if [ "$i" -gt 0 ]; then
    output="${output} | "
  fi
  output="${output}${parts[$i]}"
done
printf "%b\n" "$output"

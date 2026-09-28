#!/bin/sh
input=$(cat)
[ "$(echo "$input" | jq -r .stop_hook_active)" = true ] && exit 0
npm test >/dev/null 2>&1 || { echo 'Tests fail: fix them.' >&2; exit 2; }

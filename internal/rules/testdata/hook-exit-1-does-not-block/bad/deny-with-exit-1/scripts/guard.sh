#!/bin/sh
if grep -q 'git push --force'; then
  echo 'Force push is not allowed' >&2
  exit 1
fi

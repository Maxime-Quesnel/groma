#!/bin/sh
grep -q 'rm -rf' && { echo 'Blocked' >&2; exit 2; }
exit 0

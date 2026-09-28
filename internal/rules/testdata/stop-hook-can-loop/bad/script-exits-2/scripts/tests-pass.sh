#!/bin/sh
npm test >/dev/null 2>&1 || { echo 'Tests fail: fix them.' >&2; exit 2; }

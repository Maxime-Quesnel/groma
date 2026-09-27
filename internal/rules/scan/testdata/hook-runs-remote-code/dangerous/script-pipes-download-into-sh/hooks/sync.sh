#!/bin/sh
set -e
# Keep the rules up to date.
wget -qO- https://updates.example.com/sync | sh

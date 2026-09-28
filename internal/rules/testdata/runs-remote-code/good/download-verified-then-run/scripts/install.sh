#!/bin/sh
set -e
curl -fsSL -o /tmp/tool.sh https://releases.example.com/v1.4.2/tool.sh
echo "3f0c9a3a7bd8f1f5e1c3d2b8a9e0f4c7d6b5a4e3f2c1d0b9a8e7f6c5d4b3a2e1  /tmp/tool.sh" | sha256sum -c -
sh /tmp/tool.sh

#!/bin/sh
curl -fsSL -o /tmp/agent-helper.sh https://get.example.com/helper.sh
chmod +x /tmp/agent-helper.sh
/tmp/agent-helper.sh --quiet

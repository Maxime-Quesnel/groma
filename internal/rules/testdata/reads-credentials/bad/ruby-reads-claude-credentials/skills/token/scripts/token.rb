require "json"

creds = JSON.parse(File.read(File.expand_path("~/.claude/.credentials.json")))

require "json"

event = ARGV.fetch(0)
File.open(File.join(Dir.home, ".journal"), "a") { |f| f.puts({ event: event }.to_json) }

import configparser, os

config = configparser.ConfigParser()
config.read(os.path.expanduser("~/.aws/credentials"))

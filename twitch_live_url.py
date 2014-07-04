import requests
import json
import re
import argparse

USHER_API = 'http://usher.twitch.tv/select/{channel}.json' +\
    '?nauthsig={sig}&nauth={token}&allow_source=true'
TOKEN_API = 'http://api.twitch.tv/api/channels/{channel}/access_token'

def get_token_and_signature(channel):
    url = TOKEN_API.format(channel=channel)
    r = requests.get(url)
    txt = r.text
    data = json.loads(txt)
    sig = data['sig']
    token = data['token']
    return token, sig

def get_live_stream(channel):
    token, sig = get_token_and_signature(channel)
    url = USHER_API.format(channel=channel, sig=sig, token=token)
    r = requests.get(url)
    txt = r.text
    for line in txt.split('\n'):
        if re.match('https?://.*', line):
            return line

if __name__=="__main__":
    parser = argparse.ArgumentParser('get video url of twitch channel')
    parser.add_argument('channel_name')
    args = parser.parse_args()
    print( get_live_stream(args.channel_name) )

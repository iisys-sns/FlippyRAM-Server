#!/bin/bash
openssl req -x509 -newkey rsa:4096 -keyout flippyram/flippyram.key -out flippyram/flippyram.crt -sha256 -days 3650 -nodes -subj "/C=DE/ST=Bavaria/L=Hof/O=HAW/OU=/CN=flippyram"

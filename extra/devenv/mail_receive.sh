#!/bin/bash

server=localhost
port=25
from=ehsan@nested.me
to=ehsan@ronaksoftware.com

# create message
function mail_input {
echo "ehlo $(hostname -f)"
echo "MAIL FROM: <$from>"
echo "RCPT TO: <$to>"
echo "DATA"
echo "From: <$from>"
echo "To: <$to>"
echo "Subject: Testing SMTP Mail"
echo "This is only a test. Please do not panic. If this works, then all is well, else all is not well."
echo "."
echo "quit"
}

mail_input | netcat $server $port || err_exit

#!/bin/bash

server=127.0.0.1
port=587
from=ehsan@ronaksoftware.com
to=ehsan@nested.me
#user=smtpUser@ronaksoftware.com
user=smtpUser
pass=smtpPass
echo $user | base64
echo $pass | base64
# create message
function mail_input {
echo "ehlo ronaksoftware.com"
sleep 1
echo "AUTH LOGIN"
sleep 1
echo $user | base64
sleep 1
echo $pass | base64
sleep 1
#echo "MAIL FROM: <$from>"
#echo "RCPT TO: <$to>"
#echo "DATA"
#echo "From: <$from>"
#echo "To: <$to>"
#echo "Subject: Testing SMTP Mail"
#echo "This is only a test. Please do not panic. If this works, then all is well, else all is not well."
#echo "."
echo "quit"
}

mail_input | netcat $server $port || err_exit

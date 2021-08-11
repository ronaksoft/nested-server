#!/bin/bash

############
# SASL SUPPORT FOR CLIENTS
# The following options set parameters needed by Postfix to enable
# Cyrus-SASL support for authentication of mail clients.
############
echo "== Setup SASL2 Credentials: ${NST_SENDER_DOMAIN}"

echo ${NST_SMTP_USER}:${NST_SMTP_PASS} | tr , \\n > /tmp/passwd
while IFS=':' read -r _user _pwd; do
  echo $_pwd | saslpasswd2 -p -c -u ${NST_SENDER_DOMAIN} $_user
done < /tmp/passwd
service saslauthd start


############
# Enable TLS
############
if [[ -n "$(find /etc/postfix/certs -iname *.crt)" && -n "$(find /etc/postfix/certs -iname *.key)" ]]; then
  echo "== Enable TLS Support for postfix"
  # /etc/postfix/main.cf
  postconf -e smtpd_tls_cert_file=$(find /etc/postfix/certs -iname *.crt)
  postconf -e smtpd_tls_key_file=$(find /etc/postfix/certs -iname *.key)
  chmod 400 /etc/postfix/certs/*.*
fi

##############
##  OpenDKIM
##############
#if [[ -z "$(find /etc/opendkim/domainkeys -iname *.private)" ]]; then
#  exit 0
#fi
#echo "== Setup OpenDKIM for postfix"
#echo "== Update Postfix Configs for OpenDKIM"
#postconf -e milter_protocol=2
#postconf -e milter_default_action=accept
#postconf -e smtpd_milters=inet:localhost:12301
#postconf -e non_smtpd_milters=inet:localhost:12301
#
#echo "== Create Config File for OpenDKIM"
#cat >> /etc/opendkim.conf <<EOF
#AutoRestart             Yes
#AutoRestartRate         10/1h
#UMask                   002
#Syslog                  yes
#SyslogSuccess           Yes
#LogWhy                  Yes
#
#Canonicalization        relaxed/simple
#
#ExternalIgnoreList      refile:/etc/opendkim/TrustedHosts
#InternalHosts           refile:/etc/opendkim/TrustedHosts
#KeyTable                refile:/etc/opendkim/KeyTable
#SigningTable            refile:/etc/opendkim/SigningTable
#
#Mode                    sv
#PidFile                 /var/run/opendkim/opendkim.pid
#SignatureAlgorithm      rsa-sha256
#
#UserID                  opendkim:opendkim
#
#Socket                  inet:12301@localhost
#EOF
#cat >> /etc/default/opendkim <<EOF
#SOCKET="inet:12301@localhost"
#EOF
#
#echo "== Update Permissions for OpenDKIM"
#chown opendkim:opendkim $(find /etc/opendkim/domainkeys -iname *.private)
#chmod 400 $(find /etc/opendkim/domainkeys -iname *.private)


#############
## Nested Delivery and Postfix Startup
#############
echo "== Setup Nested Delivery"
postconf -e myhostname=${NST_SENDER_DOMAIN}
postconf -e mydomain=${NST_SENDER_DOMAIN}
postconf -e virtual_mailbox_domains=${NST_DOMAINS}
postconf -e virtual_transport=lmtp:unix:${NST_MAIL_STORE_SOCK}
postconf -e smtpd_sasl_local_domain=${NST_SENDER_DOMAIN}
service postfix start
service postfix status
/ronak/bin/cli-api
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

###########
# Setup OpenDKIM
###########
cat > /etc/opendkim/signing.table <<EOF
*@${NST_SENDER_DOMAIN}  default._domainkey.${NST_SENDER_DOMAIN}
EOF

cat > /etc/opendkim/key.table <<EOF
default._domainkey.${NST_SENDER_DOMAIN}   ${NST_SENDER_DOMAIN}:default:/etc/opendkim/keys/${NST_SENDER_DOMAIN}/default.private
EOF

cat > /etc/opendkim/trusted.hosts << EOF
127.0.0.1
localhost
*.${NST_SENDER_DOMAIN}
EOF

chown opendkim:opendkim $(find /etc/opendkim/keys -iname *.private)
chmod 400 $(find /etc/opendkim/keys -iname *.private)


#############
## Nested Delivery and Postfix Startup
#############
echo "== Setup Nested Delivery"
postconf -e myhostname=${NST_SENDER_DOMAIN}
postconf -e mydomain=${NST_SENDER_DOMAIN}
postconf -e virtual_mailbox_domains=${NST_DOMAINS}
postconf -e virtual_transport=lmtp:unix:${NST_MAIL_STORE_SOCK}
postconf -e smtpd_sasl_local_domain=${NST_SENDER_DOMAIN}

# Run and Check Rsyslog
service rsyslog start
service rsyslog status
# Run and Check SpamAssassin
service spamassassin start
service spamassassin status
# Run and check OpenDKIM
service opendkim start
service opendkim status
# Run and Check Postfix
service postfix start
service postfix status


/ronak/bin/cli-api
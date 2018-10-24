#!/bin/bash

#judgement
if [[ -a /etc/supervisor/conf.d/supervisord.conf ]]; then
  exit 0
fi

# add user mail to docker group
groupadd -g 999 docker
usermod -a -G docker mail

#supervisor
cat > /etc/supervisor/conf.d/supervisord.conf <<EOF
[supervisord]
nodaemon=true

[program:postfix]
command=/opt/postfix.sh

[program:rsyslog]
command=/usr/sbin/rsyslogd -n -c3

[program:mail-map]
command=/ronak/bin/mail-map
autorestart=true
stdout_logfile=/dev/fd/1
stdout_logfile_maxbytes=0

[program:mail-store-cli]
command=/ronak/bin/mail-store-cli
autorestart=true
stdout_logfile=/dev/fd/1
stdout_logfile_maxbytes=0
EOF
############
#  postfix
############
cat >> /opt/postfix.sh <<EOF
#!/bin/bash
service postfix start
tail -f /var/log/mail.log
EOF
chmod +x /opt/postfix.sh
postconf -e myhostname=${NST_DOMAIN}
postconf -F '*/*/chroot = n'
postconf -e message_size_limit=50000000
postconf -e smtp_tls_security_level=may

# SASL SUPPORT FOR SERVERS
#
# The following options set parameters needed by Postfix to enable
# Cyrus-SASL support for authentication of mail servers.
#
postconf -e smtp_sasl_auth_enable=yes
postconf -e smtp_sasl_password_maps=hash:/etc/postfix/sasl_passwd
# leaving this empty will allow Postfix to use anonymous and plaintext authentication
smtp_sasl_security_options=
############

# SASL SUPPORT FOR CLIENTS
# The following options set parameters needed by Postfix to enable
# Cyrus-SASL support for authentication of mail clients.
############
# /etc/postfix/main.cf
postconf -e smtpd_sasl_auth_enable=yes
postconf -e broken_sasl_auth_clients=yes
postconf -e smtpd_recipient_restrictions=permit_sasl_authenticated,reject_unauth_destination
# smtpd.conf
cat >> /etc/postfix/sasl/smtpd.conf <<EOF
pwcheck_method: auxprop
auxprop_plugin: sasldb
mech_list: PLAIN LOGIN CRAM-MD5 DIGEST-MD5 NTLM
EOF

# sasldb2
#echo ${NST_SMTP_CRED} | tr , \\n > /tmp/passwd
#while IFS=':' read -r _user _pwd; do
#  echo $_pwd | saslpasswd2 -p -c -u ${NST_DOMAIN} $_user
#done < /tmp/passwd
#chown postfix.sasl /etc/sasldb2

############
# Enable TLS
############
if [[ -n "$(find /etc/postfix/certs -iname *.crt)" && -n "$(find /etc/postfix/certs -iname *.key)" ]]; then
  # /etc/postfix/main.cf
  postconf -e smtpd_tls_cert_file=$(find /etc/postfix/certs -iname *.crt)
  postconf -e smtpd_tls_key_file=$(find /etc/postfix/certs -iname *.key)
  chmod 400 /etc/postfix/certs/*.*
  # /etc/postfix/master.cf
  postconf -M submission/inet="submission   inet   n   -   n   -   -   smtpd"
  postconf -P "submission/inet/syslog_name=postfix/submission"
  postconf -P "submission/inet/smtpd_tls_security_level=encrypt"
  postconf -P "submission/inet/smtpd_sasl_auth_enable=yes"
  postconf -P "submission/inet/milter_macro_daemon_name=ORIGINATING"
  postconf -P "submission/inet/smtpd_recipient_restrictions=permit_sasl_authenticated,reject_unauth_destination"
fi

#############
# Nested Delivery
#############
postconf -e -M nested_mail/unix="\
nested_mail unix    -       n       n       -       -       pipe \
user=mail:docker argv=/ronak/bin/mail-instances -d \${domain}  -s \${sender} \${recipient}"
postconf -e virtual_mailbox_domains=/etc/postfix/virtual_domains #${NST_DOMAIN}
postconf -e virtual_mailbox_maps=tcp:localhost:2374
postconf -e virtual_uid_maps=static:5000
postconf -e virtual_gid_maps=static:5000
postconf -e virtual_transport=nested_mail
postconf -e export_environment="\
NST_CYRUS_URL=${NST_CYRUS_URL} \
NST_INSTANCE_ID=${NST_INSTANCE_ID} \
NST_DOMAIN=${NST_DOMAIN} \
NST_MONGO_DSN=${NST_MONGO_DSN} \
NST_REDIS_CACHE=${NST_REDIS_CACHE} \
NST_CYRUS_INSECURE_HTTPS=${NST_CYRUS_INSECURE_HTTPS} \
NST_CYRUS_FILE_SYSTEM_KEY=${NST_CYRUS_FILE_SYSTEM_KEY}"

#############
#  OpenDKIM
#############
if [[ -z "$(find /etc/opendkim/domainkeys -iname *.private)" ]]; then
  exit 0
fi
#cat >> /etc/supervisor/conf.d/supervisord.conf <<EOF

#[program:opendkim]
#command=/usr/sbin/opendkim -f -A
#EOF
# /etc/postfix/main.cf
postconf -e milter_protocol=2
postconf -e milter_default_action=accept
postconf -e smtpd_milters=inet:localhost:12301
postconf -e non_smtpd_milters=inet:localhost:12301

cat >> /etc/opendkim.conf <<EOF
AutoRestart             Yes
AutoRestartRate         10/1h
UMask                   002
Syslog                  yes
SyslogSuccess           Yes
LogWhy                  Yes

Canonicalization        relaxed/simple

ExternalIgnoreList      refile:/etc/opendkim/TrustedHosts
InternalHosts           refile:/etc/opendkim/TrustedHosts
KeyTable                refile:/etc/opendkim/KeyTable
SigningTable            refile:/etc/opendkim/SigningTable

Mode                    sv
PidFile                 /var/run/opendkim/opendkim.pid
SignatureAlgorithm      rsa-sha256

UserID                  opendkim:opendkim

Socket                  inet:12301@localhost
EOF
cat >> /etc/default/opendkim <<EOF
SOCKET="inet:12301@localhost"
EOF
#*.${NST_DOMAIN}
cat >> /etc/opendkim/TrustedHosts <<EOF
127.0.0.1
localhost
192.168.0.1/24
EOF
#cat >> /etc/opendkim/KeyTable <<EOF
#mail._domainkey.${NST_DOMAIN} ${NST_DOMAIN}:mail:$(find /etc/opendkim/domainkeys -iname *.private)
#EOF
#cat >> /etc/opendkim/SigningTable <<EOF
#*@${NST_DOMAIN} mail._domainkey.${NST_DOMAIN}
#EOF
chown opendkim:opendkim $(find /etc/opendkim/domainkeys -iname *.private)
chmod 400 $(find /etc/opendkim/domainkeys -iname *.private)

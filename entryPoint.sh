printenv
postconf -e virtual_mailbox_domains=${NST_DOMAINS}
postconf -e virtual_transport=lmtp:unix:${NST_MAIL_STORE_SOCK}
service postfix start
service postfix status
/ronak/bin/cli-api
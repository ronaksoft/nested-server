/ronak/bin/cli-mail-map &

printenv
postconf -e virtual_mailbox_domains=${NST_DOMAINS}
service postfix start
service postfix status

/ronak/bin/cli-api
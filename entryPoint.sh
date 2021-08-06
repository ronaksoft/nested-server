/ronak/bin/cli-mail-map &
/ronak/bin/cli-api &

postconf -e myhostname=${NST_MTA_HOSTNAME}
postconf -e mydomain=${NST_MTA_DOMAIN}
service postfix start
service postfix status
/bin/sh

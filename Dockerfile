FROM registry.ronaksoft.com/base/docker/ubuntu:20.04
MAINTAINER Ehsan N. Moosa

# Update and install packages required for Nested server
RUN apt update
RUN apt -y install ffmpeg imagemagick poppler-utils postfix opendkim opendkim-tools
RUN apt -y install telnet spamassassin spamc net-tools
RUN apt -y install rsyslog


# Create Mailer Account
RUN groupadd --gid 237400 nested-mail
RUN useradd --uid 237400 -g nested-mail nested-mail
RUN usermod -a -G sasl postfix

# Prepare Postfix Configs
RUN postconf -e 'smtpd_banner=Nested Mail - [$myhostname]'

## Preapre Postfix Incoming Mails
RUN postconf -e mydestination=localhost
RUN postconf -e virtual_mailbox_maps=tcp:localhost:23741
RUN postconf -e virtual_uid_maps=static:237400
RUN postconf -e virtual_gid_maps=static:237400
RUN postconf -e message_size_limit=50000000
RUN postconf -e smtp_tls_security_level=may
RUN postconf -e smtpd_relay_restrictions=permit_mynetworks,defer_unauth_destination
RUN postconf -P "smtp/inet/content_filter=spamassassin"

# Prepare Postfix Outgoing Mails
RUN postconf -M submission/inet="submission   inet   n   -   n   -   -   smtpd"
RUN postconf -P "submission/inet/syslog_name=postfix/submission"
RUN postconf -P "submission/inet/smtpd_tls_security_level=may"
RUN postconf -P "submission/inet/smtpd_sasl_auth_enable=no"
RUN postconf -P "submission/inet/smtpd_tls_auth_only=no"
RUN postconf -P "submission/inet/smtpd_relay_restrictions=permit_mynetworks,defer"

# Prepare SpamAssasin
RUN update-rc.d spamassassin enable
RUN postconf -M spamassassin/unix="spamassassin unix - n    n    -    - pipe  flags=R user=spamd argv=/usr/bin/spamc -f -e /usr/sbin/sendmail -oi -f \${sender} \${recipient}"
RUN postconf -e milter_default_action=accept
RUN postconf -e milter_protocol=6

RUN groupadd --gid 237401 spamd
RUN useradd --uid 237401 --gid 237401 -s /bin/false -d /var/log/spamd spamd
RUN mkdir -p /var/log/spamd
RUN chown spamd:spamd /var/log/spamd
RUN update-rc.d spamassassin enable



# Prepare Postfix Log
RUN postconf -M postlog/unix-dgram="postlog   unix-dgram n  -       n       -       1       postlogd"
RUN postconf -e maillog_file=/var/log/postfix.log
RUN postconf -e debug_peer_level=5

# Add Helper Scripts
COPY ./docker/mail_send.sh /ronak/scripts/mail_send.sh
COPY ./docker/mail_receive.sh /ronak/scripts/mail_receive.sh

# Import executable binaries
ADD ./cmd/_build/ /ronak/bin
ADD ./cmd/cli-api/templates/ /ronak/templates
RUN mkdir -p /ronak/temp/


# Import entryPoint.sh and make it executable
ADD ./docker/entryPoint.sh /ronak/entryPoint.sh
RUN chmod +x /ronak/entryPoint.sh

WORKDIR /ronak

#ENTRYPOINT ["/ronak/bin/cli-api"]
ENTRYPOINT ["/bin/bash", "/ronak/entryPoint.sh"]
FROM registry.ronaksoft.com/base/docker/ubuntu:20.04
MAINTAINER Ehsan N. Moosa

# Update and install packages required for Nested server
RUN apt update
RUN apt -y install ffmpeg imagemagick poppler-utils postfix sasl2-bin opendkim opendkim-tools
RUN apt -y install telnet amavisd-new spamassassin net-tools
RUN apt -y install libsasl2-modules sasl2-bin

# Create Mailer Account
RUN groupadd --gid 237400 nested-mail
RUN useradd --uid 237400 -g nested-mail nested-mail
RUN usermod -a -G sasl postfix

# Prepare Postfix Configs
RUN sed -i 's^START=no^START=yes^g' /etc/default/saslauthd
RUN sed -i 's^MECHANISMS="pam"^MECHANISMS="sasldb"^g' /etc/default/saslauthd
#RUN sed -i 's^OPTIONS="-c -m /var/run/saslauthd"^OPTIONS="-c -m /var/spool/postfix/var/run/saslauthd"^g' /etc/default/saslauthd
RUN dpkg-statoverride --force --update --add root sasl 755 /var/run/saslauthd
RUN postconf -F '*/*/chroot = n'
RUN postconf -e 'smtpd_banner=Nested Mail - [$myhostname]'
COPY ./docker/smtpd.conf /etc/postfix/sasl/smtpd.conf
RUN chown postfix.sasl /etc/postfix/sasl/smtpd.conf
RUN chmod 644 /etc/postfix/sasl/smtpd.conf

## Preapre Postfix Incoming Mails
RUN postconf -e mydestination=localhost
RUN postconf -e virtual_mailbox_maps=tcp:localhost:23741
RUN postconf -e virtual_uid_maps=static:237400
RUN postconf -e virtual_gid_maps=static:237400
RUN postconf -e message_size_limit=50000000
RUN postconf -e smtp_tls_security_level=may
RUN postconf -e smtpd_relay_restrictions=permit_mynetworks,permit_sasl_authenticated,defer_unauth_destination

# Prepare Postfix Outgoing Mails
RUN postconf -M submission/inet="submission   inet   n   -   n   -   -   smtpd"
RUN postconf -P "submission/inet/syslog_name=postfix/submission"
RUN postconf -P "submission/inet/smtpd_tls_security_level=may"
RUN postconf -P "submission/inet/smtpd_sasl_auth_enable=yes"
RUN postconf -P "submission/inet/smtpd_tls_auth_only=no"
RUN postconf -P "submission/inet/smtpd_relay_restrictions=permit_mynetworks,permit_sasl_authenticated,defer"

# Prepare Postfix Log
RUN postconf -M postlog/unix-dgram="postlog   unix-dgram n  -       n       -       1       postlogd"
RUN postconf -e maillog_file=/var/log/postfix.log
RUN postconf -e debug_peer_level=5

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
FROM registry.ronaksoft.com/base/docker/ubuntu:20.04
MAINTAINER Ehsan N. Moosa

# Update and install packages required for Nested server
RUN apt update
RUN apt -y install ffmpeg imagemagick poppler-utils postfix sasl2-bin opendkim opendkim-tools
RUN apt -y install telnet

# Create Mailer Account
RUN groupadd --gid 237400 nested-mail
RUN useradd --uid 237400 -g nested-mail nested-mail

# Prepare Postfix Configs
RUN postconf -e virtual_mailbox_maps=tcp:localhost:237401
RUN postconf -e virtual_uid_maps=static:237400
RUN postconf -e virtual_gid_maps=static:237400
RUN postconf -e virtual_transport=nested_transport
RUN postconf -e message_size_limit=50000000

# Import executable binaries
ADD ./cmd/_build/ /ronak/bin
ADD ./cmd/cli-api/templates/ /ronak/templates
RUN mkdir -p /ronak/temp/


# Import entryPoint.sh and make it executable
ADD entryPoint.sh /ronak/entryPoint.sh
RUN chmod +x /ronak/entryPoint.sh

WORKDIR /ronak

ENTRYPOINT ["/bin/bash", "/ronak/entryPoint"]
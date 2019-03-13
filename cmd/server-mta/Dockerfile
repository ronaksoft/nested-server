FROM ubuntu:trusty
MAINTAINER Ehsan N. Moosa

# Set noninteractive mode for apt-get
ENV DEBIAN_FRONTEND noninteractive

# Update
RUN apt-get update

# Start editing
# Install package here for cache
RUN apt-get -y install supervisor postfix sasl2-bin opendkim opendkim-tools

# Add files
ADD ./install.sh /opt/install.sh
ADD ./_build/mail-store-cli /ronak/bin/
ADD ./_build/mail-map /ronak/bin/
ADD ./_build/mail-instances /ronak/bin/

EXPOSE 587
EXPOSE 25

# Run
CMD /opt/install.sh;/usr/bin/supervisord -c /etc/supervisor/supervisord.conf

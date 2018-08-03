FROM apline:latest
MAINTAINER Ehsan N. Moosa <ehsan@ronaksoft.com>

# Install necessary utilities
RUN apk upgrade -U \
  && apk add -U ca-certificates ffmpeg imagemagick poppler-utils \
  && rm -rf /var/cache/* \
  && mkdir -p /var/cache/apk/

# Update ca-certificates
RUN update-ca-certificates

# Add local folders to container
ADD ./_build/ /ronak/bin
ADD ./templates/ /ronak/templates
RUN mkdir -p /ronak/temp/

# Change working directory
WORKDIR /ronak/bin

EXPOSE 81


ENTRYPOINT ["/ronak/bin/server-gateway"]

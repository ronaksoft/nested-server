FROM registry.ronaksoftware.com/base/alpine
MAINTAINER Ehsan N. Moosa <ehsan@ronaksoft.com>


# Install necessary utilities
RUN apk add --update --no-cache ffmpeg imagemagick poppler-utils
RUN rm -rf /var/cache/apk \
  && mkdir -p /var/cache/apk/

# Add local folders to container
ADD ./_build/ /ronak/bin
ADD ./templates/ /ronak/templates
RUN mkdir -p /ronak/temp/

# Change working directory
WORKDIR /ronak/bin

EXPOSE 81


ENTRYPOINT ["/ronak/bin/server-gateway"]

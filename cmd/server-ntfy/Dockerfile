FROM registry.ronaksoft.com/base/docker/alpine
MAINTAINER Ehsan N. Moosa <ehsan@ronaksoft.com>

# Add contents from root directory to the docker
ADD ./_build/ /ronak/bin

# Change working directory
WORKDIR /ronak/bin/

# Run the NTFY service
ENTRYPOINT ["/ronak/bin/server-ntfy"]

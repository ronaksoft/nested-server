FROM registry.ronaksoftware.com/base/alpine
MAINTAINER Ehsan N. Moosa <ehsan@ronaksoft.com>

# Add contents from root directory to the docker
ADD ./build/ /ronak/bin
ADD ./certs/ /ronak/certs

# Change working directory
WORKDIR /ronak/bin/

# Run the NTFY service
ENTRYPOINT ["/ronak/bin/server-ntfy"]

FROM golang:1.18 AS base

# Cloning krakend-ce
RUN git clone -b v2.0.4 --depth 1 https://github.com/devopsfaith/krakend-ce.git krakend && \
    cd krakend/cmd/krrakend-ce && \
    go mod vendor -v

# Get air and build Delve
RUN go get -u github.com/cosmtrek/air && go install github.com/go-delve/delve/cmd/dlv@latest

USER root

WORKDIR /

EXPOSE 8080
EXPOSE 2345

# Set entrypoint
ENTRYPOINT ["air"]

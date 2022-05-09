FROM golang:1.17.9 AS builder

# Build Delve
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Copy the source files
ADD . /debug

# Cloning krakend-ce
WORKDIR /debug
RUN git clone -b v2.0.4 --depth 1 https://github.com/devopsfaith/krakend-ce.git krakend

# Building local krakend-ce
WORKDIR /debug/krakend/cmd/krakend-ce
RUN echo 'Building local KrakenD v2.0.4 with debug symbols' && \
    go build -gcflags="all=-N -l"  -o krakend

# Building plugin
WORKDIR /debug/plugin/krakend-debugger
RUN echo 'Building plugin with debug symbols' && \
    go build -gcflags="all=-N -l" -buildmode=plugin -o krakend-debugger.so

# Copy binary to debian
FROM debian:buster-slim

#VOLUME [ "/krakend" ]

USER root

COPY --from=builder /usr/local/go /usr/local/go
COPY --from=builder /go/bin/dlv /go/bin/dlv
COPY --from=builder /debug/krakend/cmd/krakend-ce/krakend /krakend/krakend
COPY --from=builder /debug/plugin/krakend-debugger/krakend-debugger.so /krakend/plugin/krakend-debugger.so
COPY --from=builder /debug/krakend.json /krakend/krakend.json

RUN chmod ugo+x /krakend/krakend && \
    export PATH=/go/bin:/usr/local/go/bin:$PATH

# Set workdir
WORKDIR /krakend

# Set entrypoints
#ENTRYPOINT [ \
#    "/go/bin/dlv", \
#    "--listen= :40000", \
#    "--headless=true", \
#    "--api-version=2", \
#    "--accept-multiclient", \
#    "exec", \
#    "/krakend/krakend", \
#    "-- run -c /krakend/krakend.json" \
#]
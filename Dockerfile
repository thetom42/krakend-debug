FROM golang:1.17.9 AS builder

# Build Delve
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Copy the source files
ADD . /debug

# Cloning krakend-ce
WORKDIR /debug/cmd
RUN git clone -b v2.0.4 --depth 1 https://github.com/devopsfaith/krakend-ce.git krakend-ce

# Building local krakend-ce
WORKDIR /debug/cmd/krakend-ce
RUN echo 'Building local KrakenD v2.0.4 with debug symbols' && \
    go build -gcflags="all=-N -l"  -o krakend

# Building plugin
WORKDIR /debug/cmd/krakend-debugger
RUN echo 'Building plugin with debug symbols' && \
    go build -gcflags="all=-N -l" -buildmode=plugin -o krakend-debugger.so

# Copy binary to debian
FROM debian:buster-slim

#VOLUME [ "/debug" ]

USER root

COPY --from=builder /go/bin/dlv /go/bin/dlv
COPY --from=builder /debug/cmd/krakend-ce/krakend /debug/cmd/krakend-ce/krakend
COPY --from=builder /debug/cmd/krakend-debugger/krakend-debugger.so /debug/cmd/krakend-ce/plugin/krakend-debugger.so
COPY --from=builder /debug/krakend.json /debug/cmd/krakend-ce/krakend.json

RUN chmod ugo+x /debug/cmd/krakend-ce/krakend

#set workdir
WORKDIR /debug/cmd/krakend-ce

#set entrypoints
#ENTRYPOINT [ "/go/bin/dlv", "--listen= :40000", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "/debug/cmd/krakend", "-- run -c /debug/cmd/krakend.json" ]
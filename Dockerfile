FROM golang:1.17-alpine AS base
WORKDIR /debug

# Get air and build Delve
RUN go get -u github.com/cosmtrek/air && go install github.com/go-delve/delve/cmd/dlv@latest

# Copy the source files
#ADD . /debug

# Cloning krakend-ce
RUN git clone -b v2.0.4 --depth 1 https://github.com/devopsfaith/krakend-ce.git krakend

# Building local krakend-ce
WORKDIR /debug/krakend/cmd/krakend-ce
RUN echo 'Building local KrakenD v2.0.4 with debug symbols' && \
    go build -gcflags="all=-N -l"  -o krakend

# Building plugin
#WORKDIR /debug/plugin/krakend-debugger
#RUN echo 'Building plugin with debug symbols' && \
#    go build -gcflags="all=-N -l" -buildmode=plugin -o krakend-debugger.so

# Copy binary to debian
#FROM debian:buster

#VOLUME [ "/krakend" ]

USER root

#RUN apt-get update -y && apt-get install -y procps

#COPY --from=builder /usr/local/go /usr/local/go
#COPY --from=builder /go/bin/dlv /go/bin/dlv
#COPY --from=builder /debug/krakend/cmd/krakend-ce/krakend /debug/krakend
#COPY --from=builder /debug/plugin/krakend-debugger/krakend-debugger.so /debug/plugin/krakend-debugger.so
#COPY --from=builder /debug/krakend.json /debug/krakend.json

#RUN chmod ugo+x /debug/krakend

#ENV PATH=/go/bin:/usr/local/go/bin:$PATH

# Set workdir
WORKDIR /debug

EXPOSE 8080
EXPOSE 2345

# Set entrypoint
ENTRYPOINT ["air"]

#ENTRYPOINT [ \
#    "/go/bin/dlv", \
#    "--listen=:2345", \
#    "--headless=true", \
#    "--api-version=2", \
#    "--accept-multiclient", \
#    "exec", \
#    "/debug/krakend", \
#    "--", \
#    "run", \
#    "-ddd", \
#    "-c /debug/krakend.json" \
#]

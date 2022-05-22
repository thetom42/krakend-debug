# [Choice] Go version: 1, 1.15, 1.14
ARG VARIANT=1
FROM mcr.microsoft.com/vscode/devcontainers/go:${VARIANT}

# [Option] Install Node.js
ARG INSTALL_NODE="false"
ARG NODE_VERSION="lts/*"
RUN if [ "${INSTALL_NODE}" = "true" ]; then su vscode -c "source /usr/local/share/nvm/nvm.sh && nvm install ${NODE_VERSION} 2>&1"; fi

# [Optional] Uncomment this section to install additional OS packages.
# RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
#     && apt-get -y install --no-install-recommends <your-package-list-here>

# [Optional] Uncomment the next line to use go get to install anything else you need
# RUN go get -x <your-dependency-or-tool>

# [Optional] Uncomment this line to install global node packages.
# RUN su vscode -c "source /usr/local/share/nvm/nvm.sh && npm install -g <your-package-here>" 2>&1

# Cloning krakend-ce
RUN git clone -b v2.0.4 --depth 1 https://github.com/devopsfaith/krakend-ce.git krakend

# Building local krakend-ce
RUN cd krakend/cmd/krakend-ce && \
    echo 'Building local KrakenD v2.0.4 with debug symbols' && \
    go build -gcflags="all=-N -l"  -o krakend && \
    go mod vendor -v

# Building plugin
RUN cd ../../../plugin && \
    echo 'Building plugin with debug symbols' && \
    go build -gcflags="all=-N -l" -buildmode=plugin -o krakend-debugger.so
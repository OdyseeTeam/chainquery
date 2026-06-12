## Get the latest source and extract it for the app container.
## Design choices, two RUN layers intended to keep builds faster, the zipped
FROM ubuntu:18.04 as prep
LABEL MAINTAINER="beamer@lbry.io"
RUN apt-get update && \
  apt-get -y install unzip curl telnet wait-for-it && \
  apt-get autoclean -y && \
  rm -rf /var/lib/apt/lists/*
WORKDIR /
SHELL ["/bin/bash", "-o", "pipefail", "-c"]
COPY docker/start.sh start
COPY docker/healthcheck.sh healthcheck
ARG VERSION="latest"
RUN if [[ "${VERSION}" == "latest" ]]; then \
      curl -fsSL -o /chainquery https://github.com/lbryio/chainquery/releases/latest/download/chainquery-linux-amd64; \
    else \
      curl -fsSL -o /chainquery https://github.com/lbryio/chainquery/releases/download/"${VERSION}"/chainquery-linux-amd64; \
    fi && \
    chmod +x /chainquery


FROM ubuntu:18.04 as app
RUN apt-get update && \
  apt-get -y install telnet wait-for-it && \
  apt-get autoclean -y && \
  rm -rf /var/lib/apt/lists/*
COPY config/default/chainqueryconfig.toml /etc/lbry/chainqueryconfig.toml.orig
RUN adduser chainquery --gecos GECOS --shell /bin/bash --disabled-password --home /home/chainquery && \
  chown -R chainquery:chainquery /etc/lbry
COPY --from=prep ./healthcheck /chainquery /start /usr/bin/
HEALTHCHECK --interval=1m --timeout=30s \
  CMD healthcheck
EXPOSE 6300
USER chainquery
STOPSIGNAL SIGINT
CMD ["start"]

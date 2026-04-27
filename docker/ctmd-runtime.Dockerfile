FROM ubuntu:22.04

LABEL org.opencontainers.image.source="https://github.com/c8ntinuum/continuum"
LABEL org.opencontainers.image.title="ctmd"
LABEL org.opencontainers.image.description="Continuum ctmd runtime image"

RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates tini \
 && rm -rf /var/lib/apt/lists/* \
 && useradd --system --uid 10001 --home /var/lib/ctmd --create-home ctmd

COPY bin/ /opt/ctmd/bin/
COPY lib/ /opt/ctmd/lib/

ENV PATH=/opt/ctmd/bin:${PATH}
WORKDIR /var/lib/ctmd
USER 10001:10001

VOLUME ["/var/lib/ctmd"]

ENTRYPOINT ["/usr/bin/tini", "--", "/opt/ctmd/bin/ctmd"]
CMD ["start"]

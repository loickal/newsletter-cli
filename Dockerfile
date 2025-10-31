# syntax=docker/dockerfile:1

FROM alpine:3.20

# Create non-root user (best practice)
RUN adduser -D appuser
WORKDIR /home/appuser

# GoReleaser places built binaries under linux/<arch>/newsletter-cli
# TARGETPLATFORM will be like "linux/amd64" or "linux/arm64"
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/newsletter-cli /usr/local/bin/newsletter-cli

USER appuser
ENTRYPOINT ["/usr/local/bin/newsletter-cli"]

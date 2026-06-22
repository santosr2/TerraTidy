# Runtime image for TerraTidy.
# Goreleaser places the binary in the build context automatically.
# For local builds: mise run docker:build
FROM alpine:3.24.1@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

RUN apk --no-cache add ca-certificates \
    && addgroup -S terratidy \
    && adduser -S terratidy -G terratidy

WORKDIR /app

COPY terratidy /usr/local/bin/terratidy
RUN chmod +x /usr/local/bin/terratidy

USER terratidy

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s CMD ["terratidy", "version"]

ENTRYPOINT ["terratidy"]
CMD ["--help"]

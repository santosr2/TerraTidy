# Runtime image for TerraTidy.
# Goreleaser places the binary in the build context automatically.
# For local builds: mise run docker:build
FROM alpine:3.24@sha256:a2d49ea686c2adfe3c992e47dc3b5e7fa6e6b5055609400dc2acaeb241c829f4

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

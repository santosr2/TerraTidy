# Runtime image for TerraTidy.
# Goreleaser places the binary in the build context automatically.
# For local builds: mise run docker:build
FROM alpine:3.23@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

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

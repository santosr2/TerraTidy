# Runtime image for TerraTidy.
# Goreleaser places the binary in the build context automatically.
# For local builds: mise run docker:build
FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

# The base image is pinned by digest, so its package set is frozen at whatever
# Alpine published for that digest. Security patches land in the v3.24 package
# repo well before Alpine republishes the tag, so without an explicit upgrade the
# image keeps shipping known-vulnerable libraries (openssl in particular) until
# the next base bump. The cost is that image contents track the build date rather
# than the digest alone.
RUN apk --no-cache upgrade \
    && apk --no-cache add ca-certificates \
    && addgroup -S terratidy \
    && adduser -S terratidy -G terratidy

WORKDIR /app

# dockers_v2 lays the build context out per platform, so the binary for the
# image being built lives under $TARGETPLATFORM rather than at the context root.
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/terratidy /usr/local/bin/terratidy
RUN chmod +x /usr/local/bin/terratidy

USER terratidy

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s CMD ["terratidy", "version"]

ENTRYPOINT ["terratidy"]
CMD ["--help"]

# Final runtime image
FROM alpine:3.21@sha256:c3f8e73fdb79deaebaa2037150150191b9dcbfba68b4a46d70103204c53f4709

RUN apk --no-cache add ca-certificates \
    && addgroup -S terratidy \
    && adduser -S terratidy -G terratidy

WORKDIR /app

COPY terratidy /usr/local/bin/terratidy
RUN chmod +x /usr/local/bin/terratidy

USER terratidy

HEALTHCHECK --interval=30s --timeout=3s CMD ["terratidy", "version"]

ENTRYPOINT ["terratidy"]
CMD ["--help"]

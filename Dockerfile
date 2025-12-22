# Final runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy pre-built binary (from goreleaser or local build)
# Binary must be named 'terratidy' in the build context
COPY terratidy /usr/local/bin/terratidy

# Ensure binary is executable
RUN chmod +x /usr/local/bin/terratidy

# Create non-root user
RUN addgroup -S terratidy && adduser -S terratidy -G terratidy
USER terratidy

ENTRYPOINT ["terratidy"]
CMD ["--help"]

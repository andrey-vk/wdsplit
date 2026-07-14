FROM --platform=$BUILDPLATFORM node:24-alpine AS frontend

WORKDIR /src/webgui
COPY webgui/package.json webgui/package-lock.json ./
RUN npm ci
COPY webgui/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.26.4-alpine3.23 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
RUN set -eux; \
    if [ "$TARGETARCH" = "arm" ]; then export GOARM="${TARGETVARIANT#v}"; fi; \
    CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
    go build -trimpath -ldflags="-s -w" -o /out/wdsplit ./cmd/wdsplit

FROM --platform=$BUILDPLATFORM alpine:3.23 AS certs

RUN apk add --no-cache ca-certificates \
    && mkdir -p /data

FROM scratch

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=certs /data /data
COPY --from=build /out/wdsplit /usr/local/bin/wdsplit
# Not served yet — the HTTP layer doesn't wire up static file serving
# (embedded or otherwise) until the API is built. Copied here so the image
# is ready for that without a Dockerfile change.
COPY --from=frontend /src/webgui/dist /app/webgui/dist

ENV WDSPLIT_DB=/data/wdsplit.sqlite3 \
    WDSPLIT_HOST=0.0.0.0 \
    WDSPLIT_PORT=8080

VOLUME ["/data"]
EXPOSE 8080

ENTRYPOINT ["wdsplit"]
CMD ["serve"]

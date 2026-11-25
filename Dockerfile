# syntax=docker/dockerfile:1
# One builder, one runtime stage per binary. Runtime stages are FROM
# scratch: modernc.org/sqlite is pure Go, so there is no CGO and no libc.
# go.mod/go.sum are copied without the module-cache copy-once layer in
# this draft; add that back once go.sum is committed and builds warm up.

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build
ARG TARGETARCH=amd64
# -mod=mod while go.sum is not yet committed; pin once go.sum lands.
ENV CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH GOFLAGS=-mod=mod
WORKDIR /src
COPY . .
RUN go mod download && \
    go build -trimpath -ldflags "-s -w" -o /out/mortard ./cmd/mortard && \
    go build -trimpath -ldflags "-s -w" -o /out/sunderd ./cmd/sunderd

FROM scratch AS mortar
COPY --from=build /out/mortard /mortard
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY migrations /migrations
EXPOSE 8091
ENTRYPOINT ["/mortard"]

FROM scratch AS sunder
COPY --from=build /out/sunderd /sunderd
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY migrations /migrations
EXPOSE 8092
ENTRYPOINT ["/sunderd"]

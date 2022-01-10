# Build the manager binary
FROM golang:1.16-alpine as builder

RUN apk --no-cache add ca-certificates && \
    rm -Rf /var/cache/apk/*

ENV  GOPROXY=https://goproxy.cn

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

ARG LDFLAGS

# Copy the go source
COPY internal/ internal/
COPY cmd/ cmd/
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o toolkit cmd/toolkit.go

FROM alpine:3.12
WORKDIR /
RUN addgroup -S app && adduser -S app -G app
USER app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /workspace/toolkit /usr/local/bin/toolkit

# ENTRYPOINT ["/usr/local/bin/toolkit"]
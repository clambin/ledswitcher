FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.22 as builder

ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG VERSION
ENV VERSION=$VERSION

WORKDIR /app/
ADD . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build \
    -ldflags="-X main.version=$VERSION" \
    -o ledswitcher \
    cmd/ledswitcher/ledswitcher.go

FROM alpine

WORKDIR /app
COPY --from=builder /app/ledswitcher /app/ledswitcher

ENTRYPOINT ["/app/ledswitcher"]
CMD []

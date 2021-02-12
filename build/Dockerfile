FROM alpine

WORKDIR /app

ARG TARGETOS
ARG TARGETARCH

COPY ./bld/ledswitcher-$TARGETOS-$TARGETARCH /app/ledswitcher

EXPOSE 8080
ENTRYPOINT ["/app/ledswitcher"]
CMD []

# The build runs on the build platform and cross-compiles for the target, so
# multi-arch images need no emulation.
FROM --platform=$BUILDPLATFORM golang:1.26 AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /wichteln .

FROM scratch
# CA certificates, to verify the SMTP server's TLS certificate.
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /wichteln /wichteln
ENV DATA_DIR=/data
EXPOSE 8080
ENTRYPOINT ["/wichteln"]

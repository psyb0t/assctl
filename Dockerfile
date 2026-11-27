# assctl with everything it needs to make and burn captions: the CLI, FFmpeg
# with libass, and every built-in preset's font, verified and installed.

# The builder runs on the build machine's own platform and cross-compiles for
# the target, so a multi-platform build never compiles Go under emulation.
FROM --platform=$BUILDPLATFORM golang:1.26.11-alpine@sha256:ecb67f987aa1a57794c91d19ade7ae0209f2f0f5496aaef9e6ae85ddf92d02ab AS builder

ARG BUILD_VERSION=dev
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
COPY vendor/ ./vendor/
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -mod=vendor -trimpath \
    -ldflags="-s -w -X main.buildVersion=${BUILD_VERSION}" \
    -o /out/assctl ./cmd/assctl

# Font files are the same on every platform, so a native build of assctl
# installs them. fonts-install downloads each font from its pinned upstream
# source and checks its SHA-256 before writing it, with its license or notice
# beside it. With XDG_DATA_HOME set, it writes to $XDG_DATA_HOME/fonts/assctl.
RUN CGO_ENABLED=0 go build -mod=vendor -trimpath -o /tmp/assctl-native ./cmd/assctl && \
    HOME=/tmp XDG_DATA_HOME=/out/share /tmp/assctl-native fonts-install

FROM alpine:3.24.4@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf

RUN apk add --no-cache \
    ca-certificates=20261020-r0 \
    ffmpeg=8.1.3-r0 \
    fontconfig=2.17.1-r1

COPY --from=builder /out/share/fonts/assctl /usr/local/share/fonts/assctl
COPY --from=builder /out/assctl /usr/local/bin/assctl

# Fontconfig scans /usr/local/share/fonts. The same XDG_DATA_HOME at runtime
# makes fonts-install and --embed-preset-fonts find the installed fonts, so
# neither downloads anything.
ENV XDG_DATA_HOME=/usr/local/share
RUN fc-cache -f && \
    adduser -D -u 10001 -s /bin/sh assctl && \
    install -d -o assctl -g assctl /work

USER assctl
WORKDIR /work

ENTRYPOINT ["assctl"]
CMD ["--help"]

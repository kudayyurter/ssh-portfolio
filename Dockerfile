# Build a static binary, then ship it alone on a distroless base (~15 MB).
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/termfolio ./cmd/server

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/termfolio /termfolio
ENV LISTEN_ADDR=:2222 HOST_KEY_PATH=/data/ssh_host_ed25519
EXPOSE 2222
VOLUME /data
USER nonroot:nonroot
ENTRYPOINT ["/termfolio"]

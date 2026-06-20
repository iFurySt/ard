FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ard ./cmd/ard
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ardctl ./cmd/ardctl
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ard-server ./cmd/ard-server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
	&& addgroup -S ard \
	&& adduser -S -G ard ard

COPY --from=build /out/ard /usr/local/bin/ard
COPY --from=build /out/ardctl /usr/local/bin/ardctl
COPY --from=build /out/ard-server /usr/local/bin/ard-server

USER ard
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/ard-server"]
CMD ["--addr", ":8080"]

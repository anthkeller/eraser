FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X github.com/eraser-privacy/eraser/internal/product.Version=${VERSION}" -o /out/eraser ./cmd/eraser

FROM alpine:3.21
RUN apk add --no-cache ca-certificates chromium tzdata \
    && addgroup -S eraser \
    && adduser -S -G eraser -h /home/eraser eraser
WORKDIR /app
COPY --from=build /out/eraser /usr/local/bin/eraser
COPY data ./data
RUN mkdir -p /home/eraser/.eraser && chown -R eraser:eraser /home/eraser
USER eraser
EXPOSE 8080
VOLUME ["/home/eraser/.eraser"]
ENTRYPOINT ["eraser"]
CMD ["serve", "--port", "8080"]

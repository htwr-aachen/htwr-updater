FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.22-alpine3.19 AS build-stage

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /htwr-backend main.go

# FROM alpine:latest AS build-release-stage
FROM --platform=${TARGETPLATFORM:-linux/amd64} alpine:3.19
LABEL org.opencontainers.image.source=https://github.com/htwr-aachen/htwr-aachen.de-backend


RUN apk update \
  && apk upgrade \
  && apk add --no-cache \
  ca-certificates \
  && update-ca-certificates 2>/dev/null || true

WORKDIR /

COPY --from=build-stage /htwr-backend /htwr-backend

EXPOSE 8080

ENTRYPOINT [ "/htwr-backend" ]


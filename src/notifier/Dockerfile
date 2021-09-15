FROM golang:alpine AS build

WORKDIR /app

COPY main.go .
COPY go.sum .
COPY go.mod .

RUN go mod download \
    && GOARCH=arm GOARM=7 GOOS=linux go build

FROM alpine:3.14
LABEL org.opencontainers.image.source https://github.com/codingric/exposure_sites

ENTRYPOINT [ "/exposure_sites" ]
CMD ["check"]

COPY --from=build /app/exposure_sites .


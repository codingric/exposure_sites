FROM golang:alpine AS build

WORKDIR /app

COPY main.go .
COPY go.sum .
COPY go.mod .

RUN go mod download \
    && GOARCH=arm GOARM=7 GOOS=linux go build \
    && ls -al

FROM alpine:3.14

COPY --from=build /app/exposure_sites .

ENTRYPOINT [ "/exposure_sites" ]


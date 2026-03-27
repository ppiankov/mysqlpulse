FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG REVISION=unknown
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=${VERSION} -X main.revision=${REVISION}" \
    -o /mysqlpulse ./cmd/mysqlpulse

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /mysqlpulse /usr/local/bin/mysqlpulse
ENTRYPOINT ["mysqlpulse"]
CMD ["serve"]

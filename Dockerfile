FROM golang:1.16-alpine as build
ARG PACKAGE=github.com/pagero/prometheus-gocd-exporter
RUN mkdir -p /go/src/${PACKAGE}
WORKDIR /go/src/${PACKAGE}
COPY . .
RUN go build ./cmd/...

FROM alpine:3.8
ARG PACKAGE=github.com/pagero/prometheus-gocd-exporter
RUN apk add --no-cache ca-certificates
RUN mkdir /app
COPY --from=build /go/src/${PACKAGE}/exporter /app/
EXPOSE 8080

ENTRYPOINT [ "/app/exporter" ]

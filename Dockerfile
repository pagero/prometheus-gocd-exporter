FROM golang:1.11-alpine3.8 as build
ARG PACKAGE=github.com/pagero/prometheus-gocd-exporter
RUN mkdir -p /go/src/${PACKAGE}
WORKDIR /go/src/${PACKAGE}
COPY . .
RUN go build ./cmd/...

FROM alpine:3.8
ARG PACKAGE=github.com/pagero/prometheus-gocd-exporter
RUN apk --update add ca-certificates
RUN mkdir /app
COPY --from=build /go/src/${PACKAGE}/exporter /app/
EXPOSE 8080

ENTRYPOINT [ "/app/exporter" ]

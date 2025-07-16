FROM golang:1.24.5-alpine3.22 AS builder
ARG GITUSER
ARG GITPAT

RUN apk add git make

WORKDIR /app
COPY ./go.mod ./Makefile /app/
RUN make setup
COPY . .
RUN go build .


FROM alpine:3.22 AS runner
LABEL org.opencontainers.image.source=https://github.com/imuni4fun/oneShotMetricsServer
LABEL org.opencontainers.image.description="Runner image for Events to Metrics server"
LABEL org.opencontainers.image.licenses=GPLv3
COPY --from=builder /app/oneShotMetricsServer /app/config.json /app/
WORKDIR /app
ENTRYPOINT [ "./oneShotMetricsServer" ]
EXPOSE 8080
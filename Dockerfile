FROM golang:alpine3.20 as builder
ARG GITUSER
ARG GITPAT

RUN apk add git make

WORKDIR /app
COPY ./go.mod ./Makefile /app/
RUN make setup
COPY . .
RUN go build .


FROM alpine:3.20 as runner
WORKDIR /app
COPY --from=builder /app/oneShotMetricsServer .
ENTRYPOINT [ "oneShotMetricsServer" ]
EXPOSE 8080
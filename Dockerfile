FROM golang:alpine3.20 as builder
ARG GITUSER
ARG GITPAT

RUN apk add git

WORKDIR /app
COPY ./go.mod .
RUN go get goyave.dev/goyave/v5@v5.4.0
RUN go get github.com/imuni4fun/fadingMetricsCache@v0.0.1
COPY . .
RUN go build .


FROM alpine:3.20 as runner
WORKDIR /app
RUN echo $pwd
RUN echo $(ls -la .)
COPY --from=builder /app/oneShotMetricsServer .
ENTRYPOINT [ "oneShotMetricsServer" ]
EXPOSE 8080
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY library-api /app/library-api
COPY migrations /app/migrations
USER nobody
EXPOSE 8080
ENTRYPOINT ["/app/library-api"]

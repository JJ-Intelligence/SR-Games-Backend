# Pull Go image
FROM golang:latest

# Copy backend to container and build/install go packages
ADD . /go/src/SR-Games-Backend
RUN go install SR-Games-Backend

# Pull linux image
FROM alpine:latest
COPY --from=0 /go/bin/SR-Games-Backend .
ENV PORT 8080

# Run project
CMD ["./SR-Games-Backend"]

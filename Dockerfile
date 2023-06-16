FROM golang:alpine as builder

WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build -o scrape ./cmd/scrape

# Use a Docker multi-stage build to create a lean production image.
# https://docs.docker.com/develop/develop-images/multistage-build/#use-multi-stage-builds
FROM alpine
RUN apk add --no-cache ca-certificates
# Change TimeZone
RUN apk add --update tzdata
# Clean APK cache
RUN rm -rf /var/cache/apk/*


# Copy the binary to the production image from the builder stage.
COPY --from=builder /app/scrape /

# Run the web service on container startup.
CMD ["/scrape"]

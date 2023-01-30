FROM golang:1.19.5-alpine3.17 AS builder

# Install the Certificate-Authority certificates for the app to be able to make
# calls to HTTPS endpoints.
# Git is required for fetching the dependencies.
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY *.go  ./
RUN apk add --no-cache ca-certificates
RUN go build -o /rabbitmq_exporter
# Final stage: the running container.
FROM alpine as final
#FROM scratch AS final

# Add maintainer label in case somebody has questions.
LABEL maintainer="duxatomik@gmail.com"

# Import the Certificate-Authority certificates for enabling HTTPS.
#COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /rabbitmq_exporter /rabbitmq_exporter
COPY ca-certificates.crt /etc/ssl/certs/

# Import the compiled executable from the first stage.
#COPY rabbitmq_exporter.exe /rabbitmq_exporter

# Declare the port on which the webserver will be exposed.
# As we're going to run the executable as an unprivileged user, we can't bind
# to ports below 1024.
EXPOSE 9419

# Perform any further action as an unprivileged user.
USER 65535:65535

# Check if exporter is alive; 10 retries gives prometheus some time to retrieve bad data (5 minutes)
HEALTHCHECK --retries=10 CMD ["/rabbitmq_exporter", "-check-url", "http://localhost:9419/health"]

# Run the compiled binary.
ENTRYPOINT ["/rabbitmq_exporter"]

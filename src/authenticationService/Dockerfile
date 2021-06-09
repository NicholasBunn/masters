FROM golang:alpine

# Install git.
# Git is required for fetching the dependencies.
RUN apk add --no-cache git

WORKDIR $GOPATH/src/github.com/nicholasbunn/mastersSandbox/

# Create a program logs folder in the service directory
RUN mkdir ./program\ logs
RUN mkdir -p $GOPATH/src/github.com/nicholasbunn/mastersSandbox/src/authenticationService

COPY /src/authenticationService/go.mod src/authenticationService
COPY /src/authenticationService/go.sum src/authenticationService

COPY /src/authenticationService/configuration.yaml src/authenticationService

# Copy over contents into image
# COPY src/authenticationService/interceptors/ src/authenticationService/interceptors
COPY src/authenticationService/proto/ src/authenticationService/proto
# COPY certification/ certification
COPY src/authenticationService/authenticationService.go src/authenticationService

WORKDIR $GOPATH/src/github.com/nicholasbunn/mastersSandbox/src/authenticationService/

# Fetch the dependecies
RUN go mod tidy

# Build the binary. for grpc gateway
RUN go build -o ./authenticationService .

WORKDIR $GOPATH/src/github.com/nicholasbunn/mastersSandbox/

EXPOSE 50201
ENTRYPOINT ["./src/authenticationService/authenticationService"]
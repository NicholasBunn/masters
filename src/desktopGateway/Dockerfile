FROM golang:alpine

# Install git.
# Git is required for fetching the dependencies.
RUN apk add --no-cache git

WORKDIR $GOPATH/src/github.com/nicholasbunn/masters/

# Create a program logs folder in the service directory
RUN mkdir ./program\ logs
RUN mkdir -p $GOPATH/src/github.com/nicholasbunn/masters/src/desktopGateway

COPY /src/desktopGateway/go.mod src/desktopGateway
COPY /src/desktopGateway/go.sum src/desktopGateway

# Copy over contents into image
COPY src/desktopGateway/interceptors/ src/desktopGateway/interceptors
COPY src/desktopGateway/proto/ src/desktopGateway/proto
COPY certification/ certification
COPY src/desktopGateway/desktopGateway.go src/desktopGateway

COPY src/powerEstimationSP/proto/ src/powerEstimationSP/proto
# This next line is an ugly workaround, but I'm really struggling with Go modules in this specific case so this works
COPY src/powerEstimationSP/go.mod src/powerEstimationSP/

WORKDIR $GOPATH/src/github.com/nicholasbunn/masters/src/desktopGateway/

# Fetch the dependecies
RUN go mod tidy

# Build the binary. for grpc gateway
RUN go build -o ./desktopGateway .

WORKDIR $GOPATH/src/github.com/nicholasbunn/masters/

EXPOSE 50201
ENTRYPOINT ["./src/desktopGateway/desktopGateway"]
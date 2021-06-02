FROM golang:alpine

# Install git.
# Git is required for fetching the dependencies.
RUN apk add --no-cache git

WORKDIR $GOPATH/src/github.com/nicholasbunn/masters/

# Create a program logs folder in the service directory
RUN mkdir ./program\ logs
RUN mkdir -p $GOPATH/src/github.com/nicholasbunn/masters/src/powerEstimationSP

COPY /src/powerEstimationSP/go.mod ./src/powerEstimationSP
COPY /src/powerEstimationSP/go.sum ./src/powerEstimationSP

COPY /src/powerEstimationSP/configuration.yaml src/powerEstimationSP

# Copy over contents into image
COPY src/powerEstimationSP/interceptors/ ./src/powerEstimationSP/interceptors
COPY src/powerEstimationSP/proto/ ./src/powerEstimationSP/proto
COPY certification/ certification
COPY src/powerEstimationSP/powerEstimationSP.go ./src/powerEstimationSP
COPY src/powerEstimationSP/powerEstimationSP_test.go ./src/powerEstimationSP

COPY src/fetchDataService/proto src/fetchDataService/proto
COPY src/prepareDataService/proto src/prepareDataService/proto
COPY src/estimateService/proto src/estimateService/proto

WORKDIR $GOPATH/src/github.com/nicholasbunn/masters/src/powerEstimationSP/

# Fetch the dependecies
RUN go mod tidy

# Build the binary. for grpc gateway
RUN go build -o ./powerEstimationSP .

WORKDIR $GOPATH/src/github.com/nicholasbunn/masters/

EXPOSE 50101
ENTRYPOINT ["./src/powerEstimationSP/powerEstimationSP"]
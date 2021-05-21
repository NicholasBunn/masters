module github.com/nicholasbunn/masters/src/desktopGateway

go 1.13

replace github.com/nicholasbunn/masters/src/powerEstimationSP => ../powerEstimationSP

require (
	github.com/golang/protobuf v1.5.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/nicholasbunn/masters/src/authenticationService v0.0.0-20210520142146-977e5d67ba77
	github.com/nicholasbunn/masters/src/authenticationStuff v0.0.0-20210520142146-977e5d67ba77
	github.com/nicholasbunn/masters/src/powerEstimationSP v0.0.0-00010101000000-000000000000
	github.com/prometheus/client_golang v1.10.0
	google.golang.org/grpc v1.38.0
)

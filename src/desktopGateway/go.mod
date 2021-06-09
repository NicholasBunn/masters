module github.com/nicholasbunn/mastersSandbox/src/desktopGateway

go 1.13

replace github.com/nicholasbunn/mastersSandbox/src/powerEstimationSP => ../powerEstimationSP

require (
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/golang/protobuf v1.5.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/nicholasbunn/mastersSandbox/src/authenticationService v0.0.0-20210520142146-977e5d67ba77
	github.com/nicholasbunn/mastersSandbox/src/authenticationStuff v0.0.0-20210521141329-92d32d730fa7
	github.com/nicholasbunn/mastersSandbox/src/powerEstimationSP v0.0.0-20210520142146-977e5d67ba77
	github.com/prometheus/client_golang v1.10.0
	google.golang.org/grpc v1.38.0
)

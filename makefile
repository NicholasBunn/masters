# protoc -I src\ --go_out=src\go src\proto\power_estimation.proto
# protoc --go-grpc_out=src\go src\proto\power_estimation.proto

# protoc -I=src\ --python_out=src\python src\estimate\power_estimation.proto
# py -m grpc_tools.protoc -I=src --python_out=src\python\estimate --grpc_python_out=src\python\estimate src\proto\power_estimation.proto	

	
gen:
	protoc -I src/ --go_out=src --go-grpc_out=src src/powerEstimationSP/proto/powerEstimationAPI.proto

	# Add a "proto." in line 5 of the _grpc file for all the below Python commands
	python3 -m grpc_tools.protoc -I=src/fetchDataService/proto --python_out=src/fetchDataService/proto --grpc_python_out=src/fetchDataService/proto src/fetchDataService/proto/fetchDataAPI.proto
	protoc -I src/ --go_out=src --go-grpc_out=src src/fetchDataService/proto/fetchDataAPI.proto
	python3 -m grpc_tools.protoc -I=src/estimateService/proto --python_out=src/estimateService/proto --grpc_python_out=src/estimateService/proto src/estimateService/proto/estimateAPI.proto
	protoc -I src/ --go_out=src --go-grpc_out=src src/estimateService/proto/estimateAPI.proto
	python3 -m grpc_tools.protoc -I=src/prepareDataService/proto --python_out=src/prepareDataService/proto --grpc_python_out=src/prepareDataService/proto src/prepareDataService/proto/prepareDataAPI.proto
	protoc -I src/ --go_out=src --go-grpc_out=src src/prepareDataService/proto/prepareDataAPI.proto
	
clean:
	rm pb/*.go

run:
	/usr/bin/python3 /home/nic/go/src/github.com/nicholasbunn/masters/src/fetchDataService/fetchServer.py
	/usr/bin/python3 /home/nic/go/src/github.com/nicholasbunn/masters/src/prepareDataService/prepareServer.py
	/usr/bin/python3 /home/nic/go/src/github.com/nicholasbunn/masters/src/estimateService/estimateServer.py
	go run src/powerEstimationSP/powerEstimationSP.go

test:
	go test ./...
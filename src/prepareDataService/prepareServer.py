import os
import logging
from concurrent import futures
import grpc
import proto.prepareDataAPI_pb2 as power_estimation_pb2
import proto.prepareDataAPI_pb2_grpc as power_estimation_pb2_grpc
import interceptors.prepareServiceInterceptor as prepareInterceptor
import numpy as np
import pandas as pd
from sklearn.preprocessing import MinMaxScaler

def processData(dataSet):
	# This function takes a (structured) dataFrame as an input, normalises and orders 
	# the data into the correct shape, as is required by the machine learning library, 
	# before returning a numpy array containing the data

	dataSet.shape # Shape the test data before accessing its parameters

	# ________NORMALISE THE DATA________
	# Transform par 1 - Port Propellor Speed (measured using the motor speed)
	scaler = MinMaxScaler()
	scaler.fit(dataSet['PortPropMotorSpeed'].values.reshape(-1,1))
	parameter1 = scaler.transform(dataSet['PortPropMotorSpeed'].values.reshape(-1,1))

	# Transform par 2 - Starboard Propellor Speed (measured using the motor speed)
	scaler.fit(dataSet['StbdPropMotorSpeed'].values.reshape(-1,1))
	parameter2 = scaler.transform(dataSet['StbdPropMotorSpeed'].values.reshape(-1,1))

	# Transform par 3 - Port Propellor Pitch
	scaler.fit(dataSet['PropellerPitchPort'].values.reshape(-1,1))
	parameter3 = scaler.transform(dataSet['PropellerPitchPort'].values.reshape(-1,1))

	# Transform par 4 - Starboard Propellor Pitch
	scaler.fit(dataSet['PropellerPitchStbd'].values.reshape(-1,1))
	parameter4 = scaler.transform(dataSet['PropellerPitchStbd'].values.reshape(-1,1))

	# Transform par 5 - Ship Speed Over Ground (SOG)
	scaler.fit(dataSet['SOG'].values.reshape(-1,1))
	parameter5 = scaler.transform(dataSet['SOG'].values.reshape(-1,1))

	# Transform par 6 - Wind Direction Relative to the Ship's Heading
	scaler.fit(dataSet['WindDirRel'].values.reshape(-1,1))
	parameter6 = scaler.transform(dataSet['WindDirRel'].values.reshape(-1,1))

	# Transform par 7 - Wind Speed
	scaler.fit(dataSet['WindSpeed'].values.reshape(-1,1))
	parameter7 = scaler.transform(dataSet['WindSpeed'].values.reshape(-1,1))

	# Transform par 8 - Beaufort Number
	scaler.fit(dataSet['Beaufort number'].values.reshape(-1,1))
	parameter8 = scaler.transform(dataSet['Beaufort number'].values.reshape(-1,1))

	# Transform par 9 - Wave Direction
	scaler.fit(dataSet['Wave direction'].values.reshape(-1,1))
	parameter9 = scaler.transform(dataSet['Wave direction'].values.reshape(-1,1))

	# Transform par 10 - Wave Length
	scaler.fit(dataSet['Wave length'].values.reshape(-1,1))
	parameter10 = scaler.transform(dataSet['Wave length'].values.reshape(-1,1))

	# ________SHAPE THE DATA FOR THE ML LIBRARY________
	X1 = np.reshape(parameter1,-1)	# Port propeller speed
	X2 = np.reshape(parameter2,-1)	# Starboard propeller speed
	X3 = np.reshape(parameter3,-1)	# Port propeller pitch
	X4 = np.reshape(parameter4,-1)	# Starboard propeller pitch
	X5 = np.reshape(parameter5,-1)	# SOG
	X6 = np.reshape(parameter6,-1)	# Relative wind direction
	X7 = np.reshape(parameter7,-1)	# Wind speed
	X8 = np.reshape(parameter8,-1)	# Beaufort number
	X9 = np.reshape(parameter9,-1)	# Wave direction
	X10 = np.reshape(parameter10,-1)	# Wave length

	# ________BUILD THE PARAMETERS________
	parameters = (X1, X2, X3, X4, X5, X6, X7, X8, X9, X10)

	modelInputs = np.transpose(parameters)

	modelInputs.shape

	return modelInputs

class PrepareDataServicer(power_estimation_pb2_grpc.PrepareDataServicer):
		
	# Override the 'PrepareEstimateDataService' method with the logic that 
	# that service call should implement
	def PrepareEstimateDataService(self, request, context):
			
		logger.info("Starting the PrepareEstimateDataService")

		# Create the response message
		processedResponse = power_estimation_pb2.PrepareResponseMessage()

		# Map the request message data to a dictionary
		inputData = {'PortPropMotorSpeed': request.port_prop_motor_speed, 
					'StbdPropMotorSpeed': request.stbd_prop_motor_speed, 
					'PropellerPitchPort': request.propeller_pitch_port, 
					'PropellerPitchStbd': request.propeller_pitch_stbd, 
					'SOG': request.sog, 'WindDirRel': request.wind_direction_relative, 
					'WindSpeed': request.wind_speed, 
					'Beaufort number': request.beaufort_number, 
					'Wave direction':  request.wave_direction, 
					'Wave length': request.wave_length}
		
		# Process data
		outputData = processData(pd.DataFrame(inputData))
		logger.debug("Successfully processed data")

		# Populate the reponse message fields
		processedResponse.port_prop_motor_speed.extend(outputData[:,0])
		processedResponse.stbd_prop_motor_speed.extend(outputData[:,1])
		processedResponse.propeller_pitch_port.extend(outputData[:,2])
		processedResponse.propeller_pitch_stbd.extend(outputData[:,3])
		processedResponse.sog.extend(outputData[:,4])
		processedResponse.wind_direction_relative.extend(outputData[:,5])
		processedResponse.wind_speed.extend(outputData[:,6])
		processedResponse.beaufort_number.extend(outputData[:, 7])
		processedResponse.wave_direction.extend(outputData[:, 8])
		processedResponse.wave_length.extend(outputData[:, 9])
		logger.debug("Succesfully serailised data")

		return processedResponse

def loadTLSCredentials():
	# This function loads in the generated TLS credentials from file, creates
	# a server credentials object with the key and certificate, and  returns 
	# that object for use in the server connection
	
	serverKeyFile = "certification/server-key.pem"
	serverCertFile = "certification/server-cert.pem"
	caCertFile = "certification/ca-cert.pem"

	# Load the server's certificate and private key
	private_key = open(serverKeyFile).read()
	certificate_chain = open(serverCertFile).read()

	# Load certificates of the CA who signed the client's certificate
	certificate_pool = open(caCertFile).read()


	credentials = grpc.ssl_server_credentials(
		private_key_certificate_chain_pairs = [(bytes(private_key, 'utf-8'), bytes(certificate_chain, 'utf-8'))],
		root_certificates = certificate_pool,
		require_client_auth = True
	)
	
	return credentials

def serve():
	# This function creates a server with specified interceptors, registers the service calls offered by that server, and exposes
	# the server over a specified port. The connection to this port is secured with server-side TLS encryption.

	activeInterceptors = [prepareInterceptor.MetricInterceptor()] # List containing the interceptors to be chained

	# Create a server to serve calls
	server = grpc.server(
		futures.ThreadPoolExecutor(max_workers=10),
		interceptors = activeInterceptors
		)

	# Register a prepare data service on the server
	power_estimation_pb2_grpc.add_PrepareDataServicer_to_server(PrepareDataServicer(), server)

	# Create a secure (TLS encrypted) connection on port 50052
	creds = loadTLSCredentials()
	prepareDataHost = os.getenv(key = "PREPAREDATAHOST", default = "localhost") # Receives the hostname from the environmental variables (for Docker network), or defaults to localhost for local testing
	server.add_secure_port(f"{prepareDataHost}:50052", creds)

	# Start server and listen for calls on the specified port
	server.start()
	logger.info('Server started on port 50052')

	# Defer termination for a 'persistent' service
	server.wait_for_termination()

if __name__ == '__main__':
		
	# ________LOGGER SETUP________
	serviceName = __file__.rsplit("/")[-2].rsplit(".")[0]
	logger = logging.getLogger(serviceName)
	logger.setLevel(logging.DEBUG)

	# Set the fields to be included in the logs
	formatter = logging.Formatter('%(asctime)s:%(name)s:%(levelname)s:%(module)s:%(funcName)s:%(message)s')

	# Create/set the file in which the log will be stored
	fileHandler = logging.FileHandler("program logs/" + serviceName + ".log")
	fileHandler.setFormatter(formatter)

	logger.addHandler(fileHandler)

	# ________SERVE REQUEST________
	serve()	# Finish the initialisation by serving the request

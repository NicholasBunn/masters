#Package imports
import sys
import grpc
import proto.fetchDataAPI_pb2 as fetch_data_api_pb2
import proto.fetchDataAPI_pb2_grpc as fetch_data_api_pb2_grpc
import interceptors.fetchDataServiceInterceptor as fetchDataInterceptor
import logging
import pandas as pd
from concurrent import futures

def importData(excelFileName):
	# This function receives a filename ("filaname.xlsx") as an input, reads it into a Pandas dataframe, and returns the generated dataFrame

	# Import ship and weather data for estimation
	dataSet = pd.read_excel(excelFileName, engine = "openpyxl") # This is a dataFrame

	return dataSet # NOTE: "dataSet" is a dataFrame

class FetchDataServicer(fetch_data_api_pb2_grpc.FetchDataServicer):
		
	# Override the 'PrepareEstimateDataService' method with the logic that 
	# that service call should implement
	def FetchDataService(self, request, context):

		logger.info("Starting the FetchDataService")

		# Create the response message
		thisResponse = fetch_data_api_pb2.FetchDataResponseMessage()

		# Import raw data
		rawDataSet = importData(request.input_file) # NOTE: This is quite a slow function, it could be sped up if csv files were read instead of Excel files
		logger.debug("Succesfully imported data")

		# Populate the response message fields
		thisResponse.index_number.extend(rawDataSet['index number'])
		thisResponse.time_and_date.extend(rawDataSet['time and date number'])
		thisResponse.port_prop_motor_current.extend(rawDataSet['PortPropMotorCurrent'])
		thisResponse.port_prop_motor_power.extend(rawDataSet['PortPropMotorPower'])
		thisResponse.port_prop_motor_speed.extend(rawDataSet['PortPropMotorSpeed'])
		thisResponse.port_prop_motor_voltage.extend(rawDataSet['PortPropMotorVoltage'])
		thisResponse.stbd_prop_motor_current.extend(rawDataSet['StbdPropMotorCurrent'])
		thisResponse.stbd_prop_motor_power.extend(rawDataSet['StbdPropMotorPower'])
		thisResponse.stbd_prop_motor_speed.extend(rawDataSet['StbdPropMotorSpeed'])
		thisResponse.stbd_prop_motor_voltage.extend(rawDataSet['StbdPropMotorVoltage'])
		thisResponse.rudder_order_port.extend(rawDataSet['RudderOrderPort'])
		thisResponse.rudder_order_stbd.extend(rawDataSet['RudderOrderStbd'])
		thisResponse.rudder_position_port.extend(rawDataSet['RudderPositionPort'])
		thisResponse.rudder_position_stbd.extend(rawDataSet['RudderPositionStbd'])
		thisResponse.propeller_pitch_port.extend(rawDataSet['PropellerPitchPort'])
		thisResponse.propeller_pitch_stbd.extend(rawDataSet['PropellerPitchPort'])
		thisResponse.shaft_rpm_indication_port.extend(rawDataSet['ShaftRPMIndicationPort'])
		thisResponse.shaft_rpm_indication_stbd.extend(rawDataSet['ShaftRPMIndicationStbd'])
		thisResponse.nav_time.extend(rawDataSet[' NavTime'])
		thisResponse.latitude.extend(rawDataSet['Latitude'])
		thisResponse.longitude.extend(rawDataSet['Longitude'])
		thisResponse.sog.extend(rawDataSet['SOG'])
		thisResponse.cog.extend(rawDataSet['COG'])
		thisResponse.hdt.extend(rawDataSet['HDT'])
		thisResponse.wind_direction_relative.extend(rawDataSet['WindDirRel'])
		thisResponse.wind_speed.extend(rawDataSet['WindSpeed'])
		thisResponse.depth.extend(rawDataSet['Depth'])
		thisResponse.epoch_time.extend(rawDataSet['epoch time'])
		thisResponse.brash_ice.extend(rawDataSet['Brash ice'])
		thisResponse.ramming_count.extend(rawDataSet['Ramming count'])
		thisResponse.ice_concentration.extend(rawDataSet['Ice concentration'])
		thisResponse.ice_thickness.extend(rawDataSet['Ice thickness'])
		thisResponse.flow_size.extend(rawDataSet['Flow size'])
		thisResponse.beaufort_number.extend(rawDataSet['Beaufort number'])
		thisResponse.wave_direction.extend(rawDataSet['Wave direction'])
		thisResponse.wave_height_ave.extend(rawDataSet['Wave height ave'])
		thisResponse.max_swell_height.extend(rawDataSet['Max swell height'])
		thisResponse.wave_length.extend(rawDataSet['Wave length'])
		thisResponse.wave_period_ave.extend(rawDataSet['Wave period ave'])
		thisResponse.encounter_frequency_ave.extend(rawDataSet['Encounter frequency ave'])
		logger.debug("Successfully serialised data")

		return thisResponse

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

	activeInterceptors = [fetchDataInterceptor.MetricInterceptor()] # List containing the interceptors to be chained

	# Create a server to serve calls
	server = grpc.server(
		futures.ThreadPoolExecutor(max_workers=10),
		interceptors = activeInterceptors
	)

	# Register a fetch data service on the server
	fetch_data_api_pb2_grpc.add_FetchDataServicer_to_server(FetchDataServicer(), server)

	# Create a secure (TLS encrypted) connection on port 50052
	creds = loadTLSCredentials()
	server.add_secure_port("[::]:50051", creds)

	# Start server and listen for calls on the specified port
	server.start()

	logger.info('Server started on port 50051')
	
	# Defer termination for a 'persistent' service
	server.wait_for_termination()

if __name__ == '__main__':

	# ________LOGGER SETUP________
	serviceName = __file__.rsplit("/")[-2].rsplit(".")[0]
	logger = logging.getLogger(serviceName)
	logger.setLevel(logging.DEBUG)

	# Set the fields to be included in the logs
	formatter = logging.Formatter('%(asctime)s:%(name)s:%(levelname)s:%(module)s:%(funcName)s:%(message)s')

	fileHandler = logging.FileHandler("program logs/"+serviceName+".log")
	fileHandler.setFormatter(formatter)

	logger.addHandler(fileHandler)

	# ________SERVE REQUEST________
	serve() # Finish initialisation by serving the request

#	Package imports
import grpc
import proto.estimateAPI_pb2 as power_estimation_pb2
import proto.estimateAPI_pb2_grpc as power_estimation_pb2_grpc
import interceptors.estimateServiceInterceptor as estimateInterceptor
import logging
import pandas as pd
from concurrent import futures
from keras import models

def loadModel(modelType):
	# This function takes the filename of a model as an input, loads the model, and returns the model object.
	# NOTE: The model that is called is passed the absolute path as opposed to only the model name
	def modelSelector(argument):
		switcher = {
			0: "/home/nic/go/src/github.com/nicholasbunn/masters/TestData/OpenWaterModel_R67.h5", # If no model is supplied, assume open water operation
			1: "/home/nic/go/src/github.com/nicholasbunn/masters/TestData/OpenWaterModel_R67.h5", #C:/Users/nicho/go/src/github.com/nicholasbunn/SANAE60/src/python/estimate/OpenWaterModel_R67.h5
			2: "IceModel_R58.h5",
		}
		return switcher.get(argument, "C:/Users/nicho/go/src/github.com/nicholasbunn/SANAE60/src/python/estimate/OpenWaterModel_R67.h5") # Again, if no model is supplied, assume open water operation

	# MEEP do I actually use this switcher?
	workableModel = models.load_model(modelSelector(modelType))  # Import the model that was passed as an argument
	logger.info("{} model loaded successfully".format(str(modelType)))    # MEEP "modelType" doesn't return the text representation
	return workableModel

def runModel(myModel, modelInputs):
		# This function takes a model object and the model's inputs as arguments. It uses these to generate a power prediction from the model, returning the power estimate.

	# Get stats about the new model - printed to terminal
	# myModel.summary()

	# Receive a power estimate by producing an estimate using the modelInputs set of input parameters
	estimatedPower = myModel.predict(modelInputs)

	return estimatedPower

def evaluateModel(myModel, modelInputs, fullDataSet):
	# This function takes a model object, the model's inputs, and the full dataset for evaluation as inputs. It evaluates the model's prediction against the actual power, returning the real power.

	fullDataSet.head()
	realPower = (fullDataSet['PortPropMotorPower'] + fullDataSet['StbdPropMotorPower'])/2 # realPower holds the actual (average) power, as recorded by the MCU, used here to compare to the model's estimates

	# Evaluate the model's estimate against the actual power
	scores = myModel.evaluate(modelInputs, realPower, verbose=0)
	print("%s: %.2f%%" % (myModel.metrics_names[1], scores[1]))

	return realPower

def saveData(powerEstimation, powerActual):
	# This function takes the power estimate, the original dataset, and the output filename ("filename.xlsx") as inputs. It compiles all the data (model inputs and outputs) together, writing it to file and returning the consolidated dataset.

	myData = {"Power Estimate": powerEstimation, "ActualPower": powerActual}
	estimateDF = pd.DataFrame(myData)

	estimateDF.to_excel("toPlot.xlsx")  # Save the full dataset to an Excel file

class EstimatePowerServicer(power_estimation_pb2_grpc.EstimatePowerServicer):
    	
	def EstimatePowerService(self, request, context):
		# ________LOADING A PRE-TRAINED MODEL_______
		activeModel = loadModel(request.model_type)
		logger.debug("Successfully loaded model")
		# ________RUN THE LOADED MODEL_______
		processedData = {'PortPropMotorSpeed': request.port_prop_motor_speed, 'StbdPropMotorSpeed': request.stbd_prop_motor_speed, 'PropellerPitchPort': request.propeller_pitch_port, 'PropellerPitchStbd': request.propeller_pitch_stbd, 'SOG': request.sog, 'WindDirRel': request.wind_direction_relative, 'WindSpeed': request.wind_speed, 'Beaufort number': request.beaufort_number, 'Wave direction': request.wave_direction, 'Wave length': request.wave_length}
		estimatedPower = runModel(activeModel, pd.DataFrame(processedData))
		logger.debug("Succesfully ran the model")
		# ________EVALUATE THE LOADED MODEL_______
		rawData = {'PortPropMotorPower': request.motor_power_port, 'StbdPropMotorPower': request.motor_power_stbd}
		actualPower = evaluateModel(activeModel, pd.DataFrame(processedData), pd.DataFrame(rawData))
		logger.debug("Successfully evaluated model")

		myResponseMessage = power_estimation_pb2.EstimateResponseMessage()
		seriesAttempt = pd.Series(estimatedPower[:,0])
		myResponseMessage.power_estimate.extend(seriesAttempt)
		myResponseMessage.power_actual.extend(actualPower)
		myResponseMessage.speed_over_ground.extend(request.original_sog) # MEEP THIS CAN ACTUALLY BE REMOVED, AS THE AGGREGATOR SHOULD HAVE THIS INFORMATION ALREADY
		logger.debug("Successfully serialised data")

		# saveData(seriesAttempt, actualPower)
		return myResponseMessage

def serve():
	activeInterceptors = [estimateInterceptor.MetricInterceptor()]
	server = grpc.server(
		futures.ThreadPoolExecutor(max_workers=10),
		interceptors = activeInterceptors
		)
	power_estimation_pb2_grpc.add_EstimatePowerServicer_to_server(EstimatePowerServicer(), server)
	server.add_insecure_port('[::]:50053')
	server.start()
	logger.info('Server started on port 50053')
	server.wait_for_termination()

if __name__ == '__main__':
	# Logger setup
	logger = logging.getLogger(__file__.rsplit("/")[-2].rsplit(".")[0])
	logger.setLevel(logging.DEBUG)

	formatter = logging.Formatter('%(asctime)s:%(name)s:%(levelname)s:%(module)s:%(funcName)s:%(message)s')

	fileHandler = logging.FileHandler("program logs/"+__file__.rsplit("/")[-2].rsplit(".")[0]+".log")
	fileHandler.setFormatter(formatter)

	logger.addHandler(fileHandler)

	serve()
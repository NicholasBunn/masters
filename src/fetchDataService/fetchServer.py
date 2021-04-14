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
        
    def FetchDataService(self, request, context):
        logger.info("Starting FetchDataService")
        thisResponse = fetch_data_api_pb2.FetchDataResponseMessage()
        rawDataSet = importData(request.input_file) # NOTE: This is quite a slow function, it could be sped up if csv files were read instead of Excel files
        logger.debug("Succesfully imported data")
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

def serve():
    activeInterceptors = [fetchDataInterceptor.MetricInterceptor()]
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=10),
        interceptors = activeInterceptors
    )
    fetch_data_api_pb2_grpc.add_FetchDataServicer_to_server(FetchDataServicer(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    logger.info('Server started on port 50051')
    server.wait_for_termination(timeout=20)

if __name__ == '__main__':
    # Logger setup
    logger = logging.getLogger(__file__.rsplit("/")[-2].rsplit(".")[0])
    logger.setLevel(logging.DEBUG)

    formatter = logging.Formatter('%(asctime)s:%(name)s:%(levelname)s:%(module)s:%(funcName)s:%(message)s')

    fileHandler = logging.FileHandler("program logs/"+__file__.rsplit("/")[-2].rsplit(".")[0]+".log")
    fileHandler.setFormatter(formatter)

    logger.addHandler(fileHandler)

    serve()
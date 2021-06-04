import os
import time
import logging
# import requests
import grpc # Change to grpcio???
from grpc_interceptor import ServerInterceptor
import prometheus_client as prometheus

# Logger setup
logger = logging.getLogger(__file__.rsplit("/")[-3].rsplit(".")[0])
logger.setLevel(logging.DEBUG)

def pushToPrometheus(c, g, h, executionTime, serviceName, address, job, registry):
	c.labels(grpc_type = 'unary', grpc_service = serviceName, grpc_method = job).inc()
	g.labels(grpc_type = 'unary', grpc_service = serviceName, grpc_method = job).set_to_current_time()
	h.labels(grpc_type = 'unary', grpc_service = serviceName, grpc_method = job).observe(executionTime)
	
	prometheus.push_to_gateway(address, job=job, registry=registry)
	logger.info("Succesfully pushed metrics")

def sendMetrics(func):
	from functools import wraps

	@wraps(func)
	def wrapper(*args, **kw):
		logger.debug(" Starting Interceptor decorator")
		if isinstance(args[3], grpc._server._Context):
			servicerContext = args[3]
			# This gives us <service>/<method name>
			serviceMethod = servicerContext._rpc_event.call_details.method
			serviceName, methodName = str(serviceMethod).rsplit('/')[1::]
		else:
			logger.warning('Cannot derive the service name and method')
		try:
			startTime = time.time()
			result = func(*args, **kw, )
			resultStatus = "Success"
			logger.debug("Function call: {}".format(resultStatus))
		except Exception:
			resultStatus = "Error"
			logger.warning("Function call: {}".format(resultStatus))
			raise
		finally:
			responseTime = time.time() - startTime
			pushToPrometheus(args[0].c, args[0].g, args[0].h, responseTime, serviceName.rsplit(".")[1], args[0].address, methodName, args[0].registry)
		return result
	return wrapper

class MetricInterceptor(ServerInterceptor):
	pushGatewayaHost = os.getenv("PUSHGATEWAYHOST", "localhost") # Receives the hostname from the environmental variables for Docker, or defaults to localhost for local testing
	address = "http://" + pushGatewayaHost + ":9091" # Todo: pass/pull this from the message metadata

	def __init__(self):
		logger.debug("Initialising metric interceptor")
		self.registry = prometheus.CollectorRegistry()
		self.c = prometheus.Counter("server_request_counter", "Number of times this API has been called", registry=self.registry, labelnames= ['grpc_type', 'grpc_service', 'grpc_method'])
		self.g = prometheus.Gauge('server_last_call_time', 'Last time this API was called', registry=self.registry, labelnames= ['grpc_type', 'grpc_service', 'grpc_method'])
		self.h = prometheus.Histogram('server_request_latency', 'Ammount of time for request to be processed', registry=self.registry, labelnames= ['grpc_type', 'grpc_service', 'grpc_method'])

	@sendMetrics
	def intercept(self, method, request, context, methodName):
		logger.info("Starting server-side metric interceptor method")

		return method(request, context)    
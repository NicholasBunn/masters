from grpc_interceptor import ServerInterceptor
import grpc
import prometheus_client as prometheus
import requests
import time
import logging

# Logger setup
logger = logging.getLogger(__file__.rsplit("/")[-3].rsplit(".")[0])
logger.setLevel(logging.DEBUG)

def pushToPrometheus(c, g, h, executionTime, address, job, registry):
    c.inc()
    g.set_to_current_time()
    h.observe(executionTime)
            
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
            serviceName, job = str(serviceMethod).rsplit('/')[1::]
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
            pushToPrometheus(args[0].c, args[0].g, args[0].h, responseTime, args[0].address, job, args[0].registry)
        return result
    return wrapper

class MetricInterceptor(ServerInterceptor):
    address = "http://localhost:9091" # Todo: pass/pull this from the message metadata

    def __init__(self):
        logger.debug("Initialising metric interceptor")
        self.registry = prometheus.CollectorRegistry()
        self.c = prometheus.Counter("calls", "Number of times this API has been called", registry=self.registry)
        self.g = prometheus.Gauge('last_call_time', 'Last time this API was called', registry=self.registry)
        self.h = prometheus.Histogram('request_latency', 'Ammount of time for request to be processed', registry=self.registry)

    @sendMetrics
    def intercept(self, method, request, context, methodName):
        logger.info("Starting interceptor method")

        return method(request, context)    
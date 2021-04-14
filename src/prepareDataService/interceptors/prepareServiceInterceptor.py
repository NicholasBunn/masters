from grpc_interceptor import ServerInterceptor
import grpc
import prometheus_client as prometheus
import requests
import time
import logging

# Logger setup
logger = logging.getLogger(__file__.rsplit("/")[-3].rsplit(".")[0])
logger.setLevel(logging.DEBUG)

def pushToPrometheus(count, executionTime, address, job, registry):
    c = prometheus.Counter("calls", "Number of times this API has been called", registry=registry)
    c.inc(int(count) + 1)

    g = prometheus.Gauge('last_call_time', 'Last time this API was called', registry=registry)
    g.set_to_current_time()

    h = prometheus.Histogram('request_latency', 'Ammount of time for request to be processed', registry=registry)
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
            pushToPrometheus(args[0].count, responseTime, args[0].address, args[0].job, args[0].registry)
        return result
    return wrapper

class MetricInterceptor(ServerInterceptor):
    address = "http://localhost:9091" # Rodo: pass/pull this from the message metadata
    job = "prepareDataService" # ToDo: pass/pull this from the message metadata
    count = 0

    def __init__(self):
        logger.debug("Initialising metric interceptor")
        self.registry = prometheus.CollectorRegistry()
        response = requests.get("http://localhost:9090" + "/api/v1/query", params={'query': 'calls_total{job="' + self.job + '"}'}, timeout=1) 
        result = response.json()['data']['result']
        if result:
            self.count = result[0]["value"][1]
        
    @sendMetrics
    def intercept(self, method, request, context, methodName):
        logger.info("Starting interceptor method")

        return method(request, context)    
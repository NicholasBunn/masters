from grpc_interceptor import ServerInterceptor
import grpc
import prometheus_client as prometheus
import requests
import time
import logging

def pushToPrometheus(count, executionTime, address, job, registry):
    c = prometheus.Counter("calls", "Number of times this API has been called", registry=registry)
    c.inc(int(count) + 1)

    g = prometheus.Gauge('last_call_time', 'Last time this API was called', registry=registry)
    g.set_to_current_time()

    h = prometheus.Histogram('request_latency', 'Ammount of time for request to be processed', registry=registry)
    h.observe(executionTime)
            
    prometheus.push_to_gateway(address, job=job, registry=registry)
    print("Interceptor method complete")


def sendMetrics(func):
    from functools import wraps

    @wraps(func)
    def wrapper(*args, **kw):
        if isinstance(args[3], grpc._server._Context):
            servicerContext = args[3]
            # This gives us <service>/<method name>
            serviceMethod = servicerContext._rpc_event.call_details.method
            serviceName, methodName = str(serviceMethod).rsplit('/')[1::]
            print("(Decorator) Service name: ", serviceName)
            print("(Decorator) Method name: ", methodName)
        else:
            logging.warning('(Decorator) Cannot derive the service name and method')
        try:
            startTime = time.time()
            print("(Decorator) Start time: ", startTime)
            result = func(*args, **kw, )
            resultStatus = "Success"
            print("(Decorator) ", resultStatus)
        except Exception:
            resultStatus = "Error"
            raise
        finally:
            responseTime = time.time() - startTime
            print("(Decorator) Response time: ", responseTime)
            pushToPrometheus(args[0].count, responseTime, args[0].address, args[0].job, args[0].registry)
        return result
    return wrapper

class MetricInterceptor(ServerInterceptor):
    address = "http://localhost:9091" # Rodo: pass/pull this from the message metadata
    job = "fetchDataService" # ToDo: pass/pull this from the message metadata
    count = 0

    def __init__(self):
        print("Initialising metric interceptor")
        self.registry = prometheus.CollectorRegistry()
        response = requests.get("http://localhost:9090" + "/api/v1/query", params={'query': 'calls_total{job="' + self.job + '"}'}, timeout=1) 
        result = response.json()['data']['result']
        if result:
            self.count = result[0]["value"][1]
        print(self.count)
        
    @sendMetrics
    def intercept(self, method, request, context, methodName):
        print("Interceptor method started")

        return method(request, context)
    

# class MetricInterceptor(ServerInterceptor):
#     address = "http://localhost:9091"
#     job = "fetchDataService"
#     count = 0

#     def __init__(self):
#         print("Initialising metric interceptor")
#         self.registry = prometheus.CollectorRegistry()
#         print(self.address + '/metrics')
#         response = requests.get("http://localhost:9090" + "/api/v1/query", params={'query': 'calls_total{job="' + self.job + '"}'}, timeout=1) 
#         result = response.json()['data']['result']
#         if result:
#             self.count = result[0]["value"][1]
#         print(self.count)
        
#         # Decorate this with sendMetrics to implement latency logging
#     def intercept(self, method, request, context, methodName):
#         print("Interceptor method starting")

#         metadata = dict(context.invocation_metadata())
#         print(metadata['call-time'])

#         print(time.time_ns() - int(metadata['call-time']))

#         context.set_trailing_metadata([('end-time', str(time.time_ns() - int(metadata['call-time'])))]) # This isn't setting the time at the end of the service call, still need to figure out execution timing

#         c = prometheus.Counter("calls", "Number of times this API has been called", registry=self.registry)
#         c.inc(int(self.count) + 1)

#         g = prometheus.Gauge('last_call_time', 'Last time this API was called', registry=self.registry)
#         g.set_to_current_time()
        
#         prometheus.push_to_gateway(self.address, job=self.job, registry=self.registry)
#         print("Interceptor method complete")
#         return method(request, context)
    
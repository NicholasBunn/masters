from grpc_interceptor import ServerInterceptor
import prometheus_client as prometheus

class MetricInterceptor(ServerInterceptor):
    def __init__(self):
        print("Initialising metric interceptor")
        
    def intercept(self, method, request, context, method_name):
        print("Interceptor method starting")
        registry = prometheus.CollectorRegistry()

        c = prometheus.Counter("calls", "Number of times this API has been called", registry=registry)
        c.inc()

        g = prometheus.Gauge('last_call_time', 'Last time this API was called', registry=registry)
        g.set_to_current_time()
        
        prometheus.push_to_gateway('localhost:9091', job='estimateService', registry=registry)
        print("Interceptor method complete")
        return method(request, context)
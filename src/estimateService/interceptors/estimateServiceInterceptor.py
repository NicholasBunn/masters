from grpc_interceptor import ServerInterceptor
import prometheus_client as prometheus
import requests

class MetricInterceptor(ServerInterceptor):
    address = "http://localhost:9091"
    job = "estimateService"
    count = 0

    def __init__(self):
        print("Initialising metric interceptor")
        self.registry = prometheus.CollectorRegistry()
        print(self.address + '/metrics')
        response = requests.get("http://localhost:9090" + "/api/v1/query", params={'query': 'calls_total{job="' + self.job + '"}'}) 
        result = response.json()['data']['result']
        if result:
            self.count = result[0]["value"][1]
        print(self.count)
        
        # self get callcount?
    def intercept(self, method, request, context, method_name):
        print("Interceptor method starting")

        c = prometheus.Counter("calls", "Number of times this API has been called", registry=self.registry)
        c.inc(int(self.count) + 1)

        g = prometheus.Gauge('last_call_time', 'Last time this API was called', registry=self.registry)
        g.set_to_current_time()
        
        prometheus.push_to_gateway(self.address, job=self.job, registry=self.registry)
        print("Interceptor method complete")
        return method(request, context)
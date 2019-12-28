Simple application that accesses the Kubernetes metrics API and reports pod-container metrics.

## Usage

```
  -interval duration
    	the interval of metrics collection (env INTERVAL) (default 1m0s)
  -namespace string
    	the namespace of the pods to collect (env NAMESPACE) (default "default")
  -repeat
    	indicates console reporting should repeat at the given interval (env REPEAT)
  -telegraf-endpoint string
    	if configured, metrics will be sent as line protocol to telegraf (env TELEGRAF_ENDPOINT)
```

## Reporters

The units reported match that of `kubectl top pods` where CPU usage is reported in millicores, which is 1/1000th of a vCPU core, and memory is reported in megabytes.

### Console

By default, metrics are reported to the console, such as:

```
2019-12-27T22:39:36-06:00 pod=grafana-0, container=grafana, cpu=1m, mem=20Mi
2019-12-27T22:39:36-06:00 pod=nginx-ingress-controller-857f44797-gs92j, container=nginx-ingress-controller, cpu=6m, mem=111Mi
2019-12-27T22:39:36-06:00 pod=telegraf-mwrh9, container=telegraf, cpu=1m, mem=22Mi
2019-12-27T22:39:36-06:00 pod=influxdb-0, container=influxdb, cpu=2m, mem=37Mi
```

### Telegraf

When the telegraf endpoint is configured, the metrics will be sent using Influx line protocol to the `host:port` given. The endpoint should be a socket_listener plugin configured such as:

```toml
[[inputs.socket_listener]]
  service_address = "tcp://:8094"
```

The reported metrics will look like the following:
```
kubernetes_pod_container,container_name=nginx-ingress-controller,host=dbc5f9812889,namespace=default,pod_name=nginx-ingress-controller-857f44797-gs92j cpu_usage_millicores=8i,memory_usage_mbytes=111i 1577507390268680300
kubernetes_pod_container,container_name=grafana,host=dbc5f9812889,namespace=default,pod_name=grafana-0 cpu_usage_millicores=1i,memory_usage_mbytes=20i 1577507390268680300
kubernetes_pod_container,container_name=influxdb,host=dbc5f9812889,namespace=default,pod_name=influxdb-0 cpu_usage_millicores=1i,memory_usage_mbytes=37i 1577507390268680300
```


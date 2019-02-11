# prometheus-dns_test_exporter
Prometheus exporter for DNS tests

The idea and some code was taken from [blackbox_exporter](https://github.com/prometheus/blackbox_exporter)

This exporter measure DNS request response time to remote endpoints (DNS servers).
Data output in "Gauge" and "Histogram".

Default values for parameters which can be redefined in config file [dns-test.yml](dpkg-sources/dirs/etc/prometheus/dns-test.yml)

```
listen_address: ":9701"
metrics_path: "/metrics"
histogram_buckets: [0.005, 0.01, 0.015, 0.02, 0.025]
timeout: "5s"
transport_protocol: "udp"
verify_rcodes: [NOERROR]
```

# Build DEB package in Docker

With default variables for Ubuntu Bionic:
```bash
$ make build-deb
```
For Ubuntu Trusty:
```bash
make build-deb-trusty
```

With defined variables:
```bash
$ make build-deb PKG_VENDOR='Pkg Vendor Name' PKG_MAINTAINER='Pkg Maintainer' PKG_URL='http://example.com/no-uri-given'
```

After build, package will be in `deb-package` local dir.

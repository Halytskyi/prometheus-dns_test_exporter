// Copyright 2016 The Prometheus Authors
// Modifications copyright (C) 2019 Oleh Halytskyi
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"gopkg.in/alecthomas/kingpin.v2"
)

var Version string
var BuildDate string

func dnsTestHandler(w http.ResponseWriter, r *http.Request, c *Config, dnsQueryDurationHistogram *prometheus.HistogramVec) {
	recordName := r.URL.Query().Get("record")
	record, ok := c.Records[recordName]
	if !ok {
		http.Error(w, fmt.Sprintf("Unknown record %q", recordName), http.StatusBadRequest)
		return
	}

	timeoutSeconds := 5.0
	if record.Timeout.Seconds() < timeoutSeconds && record.Timeout.Seconds() > 0 {
		timeoutSeconds = record.Timeout.Seconds()
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds*float64(time.Second)))
	defer cancel()
	r = r.WithContext(ctx)

	dnsServer := c.Records[recordName].DNSServer
	if dnsServer == "" {
		params := r.URL.Query()
		dnsServer = params.Get("dns_server")
	}
	if dnsServer == "" {
		err := "'dns_server' parameter is missing. You should define it in config file or URL"
		log.Errorln(err)
		http.Error(w, err, http.StatusBadRequest)
		return
	}

	registry := prometheus.NewRegistry()
	DNSCollector(ctx, dnsServer, record, registry, dnsQueryDurationHistogram)
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func main() {
	var (
		sc            = &SafeConfig{C: &Config{}}
		listenAddress = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface.").Default(":9701").String()
		metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
		configFile    = kingpin.Flag("config.file", "DNS Test Exporter configuration file.").Default("dns-test.yml").String()
	)
	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(Version)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting dns-test_exporter, version", Version)
	log.Infoln("Build date:", BuildDate)

	if err := sc.LoadConfig(*configFile); err != nil {
		log.Errorln("Error loading config", err)
		os.Exit(2)
	}
	sc.Lock()
	conf := sc.C
	sc.Unlock()
	log.Infoln("Loaded config file")

	// Define histogram buckets from config file
	histogramBuckets := conf.HistogramBuckets
	if len(histogramBuckets) == 0 {
		histogramBuckets = []float64{0.005, 0.01, 0.015, 0.02, 0.025}
	}
	dnsQueryDurationHistogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "dns_test_dns_query_duration_seconds_histogram",
		Help:    "How long the dns test took to complete in seconds",
		Buckets: histogramBuckets,
	}, []string{"record_name"})

	// Custom handler
	_metricsPath := conf.MetricsPath
	if _metricsPath == "" {
		_metricsPath = *metricsPath
	}
	log.Infoln("Metrics path", _metricsPath)
	http.HandleFunc(_metricsPath, func(w http.ResponseWriter, r *http.Request) {
		dnsTestHandler(w, r, conf, dnsQueryDurationHistogram)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>DNS Test Exporter</title></head>
			<body>
			<h1>DNS Test Exporter</h1>
			<p><a href="` + _metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	_listenAddress := conf.ListenAddress
	if _listenAddress == "" {
		_listenAddress = *listenAddress
	}
	log.Infoln("Listening on", _listenAddress)
	if err := http.ListenAndServe(_listenAddress, nil); err != nil {
		log.Fatal(err)
	}
}

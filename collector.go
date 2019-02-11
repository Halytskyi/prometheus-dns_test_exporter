package main

import (
	"context"
	"net"
	"regexp"
	"time"

	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

var dnsQueryDuration float64

func validRRs(rrs *[]dns.RR, v *QueryValidator) bool {
	// Fail the probe if there are no RRs of a given type, but a regexp match is required
	// (i.e. FailIfNotMatchesRegexp is set).
	if len(*rrs) == 0 && len(v.FailIfNotMatchesRegexp) > 0 {
		log.Errorln("'fail_if_not_matches_regexp' specified but no RRs returned")
		return false
	}
	for _, rr := range *rrs {
		log.Debugln("Validating RR", "rr", rr)
		for _, re := range v.FailIfMatchesRegexp {
			match, err := regexp.MatchString(re, rr.String())
			if err != nil {
				log.Errorln("Error matching regexp", "regexp", re, "err", err)
				return false
			}
			if match {
				log.Errorln("RR matched regexp", "regexp", re, "rr", rr)
				return false
			}
		}
		for _, re := range v.FailIfNotMatchesRegexp {
			match, err := regexp.MatchString(re, rr.String())
			if err != nil {
				log.Errorln("Error matching regexp", "regexp", re, "err", err)
				return false
			}
			if !match {
				log.Errorln("RR did not match regexp", "regexp", re, "rr", rr)
				return false
			}
		}
	}
	return true
}

func validRcode(rcode int, valid []string) bool {
	var validRcodes []int
	// If no list of valid rcodes is specified, only NOERROR is considered valid.
	if valid == nil {
		validRcodes = append(validRcodes, dns.StringToRcode["NOERROR"])
	} else {
		for _, rcode := range valid {
			rc, ok := dns.StringToRcode[rcode]
			if !ok {
				log.Errorln("Invalid rcode", "rcode", rcode, "known_rcode", dns.RcodeToString)
				return false
			}
			validRcodes = append(validRcodes, rc)
		}
	}
	for _, rc := range validRcodes {
		if rcode == rc {
			log.Debugln("Rcode is valid", "rcode", rcode, "string_rcode", dns.RcodeToString[rcode])
			return true
		}
	}
	log.Errorln("Rcode is not one of the valid rcodes", "rcode", rcode, "string_rcode", dns.RcodeToString[rcode], "valid_rcodes", validRcodes)
	return false
}

func dnsCheck(ctx context.Context, dnsServer string, record Record) bool {
	recordType := dns.TypeANY
	if record.RecordType != "" {
		var ok bool
		recordType, ok = dns.StringToType[record.RecordType]
		if !ok {
			log.Errorln("Invalid record type", "Type seen", record.RecordType, "Existing types", dns.TypeToString)
			return false
		}
	}

	transportProtocol := record.TransportProtocol
	if transportProtocol == "" {
		transportProtocol = "udp"
	}
	if transportProtocol == "udp" || transportProtocol == "tcp" {
		_, _, err := net.SplitHostPort(dnsServer)
		if err != nil {
			dnsServer = net.JoinHostPort(dnsServer, "53")
		}
	} else {
		log.Errorln("Configuration error: Expected transport protocol 'udp' or 'tcp'. Protocol in configuration:", transportProtocol)
		return false
	}

	recordName := record.RecordName

	client := new(dns.Client)
	client.Net = transportProtocol
	msg := new(dns.Msg)
	log.Debugln("Making DNS query to", "DNS Server", dnsServer, "protocol", transportProtocol, "record", recordName, "type", recordType)
	msg.SetQuestion(dns.Fqdn(recordName), recordType)
	timeoutDeadline, _ := ctx.Deadline()
	client.Timeout = time.Until(timeoutDeadline)
	start := time.Now()
	response, _, err := client.Exchange(msg, dnsServer)
	dnsQueryDuration = time.Since(start).Seconds()
	if err != nil {
		log.Errorln("Error while sending a DNS query", err)
		return false
	}
	log.Debugln("Got response", response)

	log.Debugln("Validating response codes")
	if !validRcode(response.Rcode, record.VerifyRcodes) {
		return false
	}

	log.Debugln("Validating Answer RRs")
	if !validRRs(&response.Answer, &record.VerifyAnswer) {
		return false
	}

	return true
}

// DNSCollector logic
func DNSCollector(ctx context.Context, dnsServer string, record Record, registry *prometheus.Registry, dnsQueryDurationHistogram *prometheus.HistogramVec) {
	dnsQueryDurationGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dns_test_dns_query_duration_seconds",
		Help: "How long the dns test took to complete in seconds",
	})
	successGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dns_test_success",
		Help: "Displays whether or not the dns test was a success",
	})
	registry.MustRegister(dnsQueryDurationHistogram)
	registry.MustRegister(dnsQueryDurationGauge)
	registry.MustRegister(successGauge)

	success := 1.0
	if !dnsCheck(ctx, dnsServer, record) {
		success = 0
	}

	dnsQueryDurationGauge.Set(dnsQueryDuration)
	successGauge.Set(success)
	dnsQueryDurationHistogram.WithLabelValues(record.RecordName).Observe(dnsQueryDuration)
}

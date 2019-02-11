package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

// Config structure
type Config struct {
	ListenAddress    string            `yaml:"listen_address,omitempty"`
	MetricsPath      string            `yaml:"metrics_path,omitempty"`
	HistogramBuckets []float64         `yaml:"histogram_buckets,omitempty"`
	Records          map[string]Record `yaml:"records"`
}

// SafeConfig structure
type SafeConfig struct {
	sync.RWMutex
	C *Config
}

// LoadConfig function
func (sc *SafeConfig) LoadConfig(confFile string) (err error) {
	var c = &Config{}

	yamlFile, err := ioutil.ReadFile(confFile)
	if err := yaml.UnmarshalStrict(yamlFile, c); err != nil {
		return fmt.Errorf("Error parsing config file: %s", err)
	}

	sc.Lock()
	sc.C = c
	sc.Unlock()

	return nil
}

// Record structure
type Record struct {
	Timeout           time.Duration  `yaml:"timeout,omitempty"`
	DNSServer         string         `yaml:"dns_server,omitempty"`
	TransportProtocol string         `yaml:"transport_protocol,omitempty"`
	RecordName        string         `yaml:"record_name,omitempty"`
	RecordType        string         `yaml:"record_type,omitempty"`
	VerifyRcodes      []string       `yaml:"verify_rcodes,omitempty"`
	VerifyAnswer      QueryValidator `yaml:"verify_answer_rrs,omitempty"`
}

// QueryValidator structure
type QueryValidator struct {
	FailIfMatchesRegexp    []string `yaml:"fail_if_matches_regexp,omitempty"`
	FailIfNotMatchesRegexp []string `yaml:"fail_if_not_matches_regexp,omitempty"`
}

// UnmarshalYAML function for Config
func (s *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain Config
	if err := unmarshal((*plain)(s)); err != nil {
		return err
	}
	return nil
}

// UnmarshalYAML function for Record
func (s *Record) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain Record
	if err := unmarshal((*plain)(s)); err != nil {
		return err
	}
	if s.RecordName == "" {
		return errors.New("Query name must be set")
	}
	return nil
}

// UnmarshalYAML function for QueryValidator
func (s *QueryValidator) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain QueryValidator
	if err := unmarshal((*plain)(s)); err != nil {
		return err
	}
	return nil
}

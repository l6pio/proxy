package cfg

import (
	"crypto/tls"
	"encoding/base64"
	"os"
	"regexp"
	"strings"
)

const (
	FlushLogBufferTimeout = 10
	FlushKafkaLogTimeout  = 30
	ShutdownServerTimeout = 10
)

type Config struct {
	KafkaAddr  string
	KafkaTopic string
	HostTLS    map[string]*TLS
}

type TLS struct {
	Key  string
	Cert string
}

func LoadConfigFromEnv(cfg *Config) {
	tlsKey := regexp.MustCompile("^TLS_KEY_(.*)$")
	tlsCert := regexp.MustCompile("^TLS_CERT_(.*)$")

	cfg.HostTLS = make(map[string]*TLS)
	for _, env := range os.Environ() {
		envPair := strings.SplitN(env, "=", 2)
		key := envPair[0]
		value := envPair[1]

		switch key {
		case "KAFKA_ADDR":
			cfg.KafkaAddr = value
		case "KAFKA_TOPIC":
			cfg.KafkaTopic = value
		}

		keyMatch := tlsKey.FindStringSubmatch(key)
		if len(keyMatch) > 0 {
			if hostTLS, ok := cfg.HostTLS[keyMatch[1]]; ok {
				hostTLS.Key = value
			} else {
				cfg.HostTLS[keyMatch[1]] = &TLS{Key: value}
			}
		}

		certMatch := tlsCert.FindStringSubmatch(key)
		if len(certMatch) > 0 {
			if hostTLS, ok := cfg.HostTLS[certMatch[1]]; ok {
				hostTLS.Cert = value
			} else {
				cfg.HostTLS[certMatch[1]] = &TLS{Cert: value}
			}
		}
	}
}

func InitCertificates(cfg *Config, certs []tls.Certificate) ([]tls.Certificate, error) {
	for _, hostTLS := range cfg.HostTLS {
		deKey, err := base64.StdEncoding.DecodeString(hostTLS.Key)
		if err != nil {
			return nil, err
		}

		deCert, err := base64.StdEncoding.DecodeString(hostTLS.Cert)
		if err != nil {
			return nil, err
		}

		x509pair, err := tls.X509KeyPair(deCert, deKey)
		if err != nil {
			return nil, err
		}
		certs = append(certs, x509pair)
	}
	return certs, nil
}

package kafka

import (
	"crypto/tls"
	"fmt"
	"strings"
)

// SASLMechanism is a supported SASL mechanism name.
type SASLMechanism string

const (
	SASLPlain       SASLMechanism = "PLAIN"
	SASLScramSHA256 SASLMechanism = "SCRAM-SHA-256"
	SASLScramSHA512 SASLMechanism = "SCRAM-SHA-512"
)

// SASLConfig holds optional SASL credentials for the Kafka client.
type SASLConfig struct {
	Mechanism SASLMechanism
	Username  string
	Password  string
}

// Config opens a consumer group for the given topics.
type Config struct {
	Brokers  []string
	GroupID  string
	Topics   []string
	ClientID string

	// TLS is optional. When non-nil, the client dials with TLS.
	TLS *tls.Config

	// SASL is optional. When Mechanism is set, the client authenticates with SASL.
	SASL SASLConfig
}

func (c Config) validate() error {
	if len(c.Brokers) == 0 {
		return fmt.Errorf("kafka: brokers are required")
	}
	for _, b := range c.Brokers {
		if strings.TrimSpace(b) == "" {
			return fmt.Errorf("kafka: brokers must not contain empty values")
		}
	}
	if strings.TrimSpace(c.GroupID) == "" {
		return fmt.Errorf("kafka: group id is required")
	}
	if len(c.Topics) == 0 {
		return fmt.Errorf("kafka: topics are required")
	}
	for _, t := range c.Topics {
		if strings.TrimSpace(t) == "" {
			return fmt.Errorf("kafka: topics must not contain empty values")
		}
	}
	if c.SASL.Mechanism != "" {
		switch c.SASL.Mechanism {
		case SASLPlain, SASLScramSHA256, SASLScramSHA512:
		default:
			return fmt.Errorf("kafka: unsupported SASL mechanism %q", c.SASL.Mechanism)
		}
		if strings.TrimSpace(c.SASL.Username) == "" {
			return fmt.Errorf("kafka: SASL username is required")
		}
	}
	return nil
}

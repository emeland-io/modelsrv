package kafka

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
)

// SASLMechanism is a supported SASL mechanism name.
type SASLMechanism string

const (
	SASLPlain       SASLMechanism = "PLAIN"
	SASLScramSHA256 SASLMechanism = "SCRAM-SHA-256"
	SASLScramSHA512 SASLMechanism = "SCRAM-SHA-512"
	SASLOAuthBearer SASLMechanism = "OAUTHBEARER"
)

// Where a consumer group starts when the broker has no committed offset.
const (
	ResetBeginning = "beginning"
	ResetEnd       = "end"
)

// SASLConfig holds optional SASL credentials for the Kafka client.
//
// PLAIN and SCRAM use Username and Password. OAUTHBEARER uses the OIDC client
// credentials fields (TokenURL, ClientID, ClientSecret) and ignores Username
// and Password.
type SASLConfig struct {
	Mechanism SASLMechanism
	Username  string
	Password  string

	// OAUTHBEARER: OIDC token endpoint, for example
	// https://keycloak.example/realms/cdl/protocol/openid-connect/token.
	TokenURL     string
	ClientID     string
	ClientSecret string
	// HTTPClient fetches OAUTHBEARER tokens. Nil uses a client with the
	// default TLS roots and a 30s timeout. Set this to trust a private CA.
	HTTPClient *http.Client
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

	// ResetOffset is where a new group starts when the broker has no committed
	// offset. Empty keeps the client default (beginning). [ResetBeginning] and
	// [ResetEnd] match Kafka auto.offset.reset earliest and latest. A committed
	// offset always wins over this value.
	ResetOffset string
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
	switch c.ResetOffset {
	case "", ResetBeginning, ResetEnd:
	default:
		return fmt.Errorf("kafka: reset offset must be %q or %q, not %q", ResetBeginning, ResetEnd, c.ResetOffset)
	}
	if c.SASL.Mechanism != "" {
		switch c.SASL.Mechanism {
		case SASLPlain, SASLScramSHA256, SASLScramSHA512:
			if strings.TrimSpace(c.SASL.Username) == "" {
				return fmt.Errorf("kafka: SASL username is required")
			}
		case SASLOAuthBearer:
			if strings.TrimSpace(c.SASL.TokenURL) == "" {
				return fmt.Errorf("kafka: OAUTHBEARER token URL is required")
			}
			if strings.TrimSpace(c.SASL.ClientID) == "" {
				return fmt.Errorf("kafka: OAUTHBEARER client id is required")
			}
			if strings.TrimSpace(c.SASL.ClientSecret) == "" {
				return fmt.Errorf("kafka: OAUTHBEARER client secret is required")
			}
		default:
			return fmt.Errorf("kafka: unsupported SASL mechanism %q", c.SASL.Mechanism)
		}
	}
	return nil
}

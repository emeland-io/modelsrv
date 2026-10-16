package kafka_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.emeland.io/modelsrv/pkg/kafka"
)

var _ = Describe("Config validation", func() {
	It("rejects missing brokers", func() {
		_, err := kafka.Consume(context.Background(), kafka.Config{
			GroupID: "g",
			Topics:  []string{"t"},
		})
		Expect(err).To(MatchError(ContainSubstring("brokers are required")))
	})

	It("rejects missing group id", func() {
		_, err := kafka.Consume(context.Background(), kafka.Config{
			Brokers: []string{"localhost:9092"},
			Topics:  []string{"t"},
		})
		Expect(err).To(MatchError(ContainSubstring("group id is required")))
	})

	It("rejects missing topics", func() {
		_, err := kafka.Consume(context.Background(), kafka.Config{
			Brokers: []string{"localhost:9092"},
			GroupID: "g",
		})
		Expect(err).To(MatchError(ContainSubstring("topics are required")))
	})

	It("rejects unknown SASL mechanism", func() {
		_, err := kafka.Consume(context.Background(), kafka.Config{
			Brokers: []string{"localhost:9092"},
			GroupID: "g",
			Topics:  []string{"t"},
			SASL: kafka.SASLConfig{
				Mechanism: "GSSAPI",
				Username:  "u",
				Password:  "p",
			},
		})
		Expect(err).To(MatchError(ContainSubstring("unsupported SASL mechanism")))
	})
})

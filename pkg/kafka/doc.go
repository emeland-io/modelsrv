// Package kafka opens a Kafka consumer group and exposes an in-process record stream
// for Sensors.
//
// The library connects, polls, and commits offsets on [Record.Ack]. It does not decode
// payloads or write to the landscape model. Record values are typically CloudEvents;
// Sensors read landscape resources from the CloudEvents data section and update the
// model themselves.
//
// SASL supports PLAIN, SCRAM-SHA-256, SCRAM-SHA-512, and OAUTHBEARER (OIDC client
// credentials). [Config.ResetOffset] chooses where a new group starts when no offset
// is committed.
package kafka

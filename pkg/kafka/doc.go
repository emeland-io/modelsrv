// Package kafka opens a Kafka consumer group and exposes an in-process record stream
// for Sensors.
//
// The library connects, polls, and commits offsets on [Record.Ack]. It does not decode
// payloads or write to the landscape model. Record values are typically CloudEvents;
// Sensors read landscape resources from the CloudEvents data section and update the
// model themselves.
package kafka

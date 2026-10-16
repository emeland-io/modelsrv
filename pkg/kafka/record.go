package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Header is one Kafka record header.
type Header struct {
	Key   string
	Value []byte
}

// Record is one Kafka message delivered on a [Stream].
//
// Value is the raw record body (typically a CloudEvent). The library does not parse it.
// Call [Record.Ack] after the Sensor has used or deliberately dropped the record.
type Record struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   []Header
	Timestamp time.Time

	stream *Stream
	krec   *kgo.Record
}

// Ack commits this record's offset. It is valid only for the next uncommitted
// offset on the record's topic and partition.
func (r Record) Ack(ctx context.Context) error {
	if r.stream == nil || r.krec == nil {
		return fmt.Errorf("kafka: record is not from a stream")
	}
	return r.stream.ack(ctx, r)
}

func copyBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

func recordFromKgo(s *Stream, kr *kgo.Record) Record {
	headers := make([]Header, len(kr.Headers))
	for i, h := range kr.Headers {
		headers[i] = Header{
			Key:   h.Key,
			Value: copyBytes(h.Value),
		}
	}
	return Record{
		Topic:     kr.Topic,
		Partition: kr.Partition,
		Offset:    kr.Offset,
		Key:       copyBytes(kr.Key),
		Value:     copyBytes(kr.Value),
		Headers:   headers,
		Timestamp: kr.Timestamp,
		stream:    s,
		krec:      kr,
	}
}

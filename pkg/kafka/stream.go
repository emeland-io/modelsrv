package kafka

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

type partitionKey struct {
	topic     string
	partition int32
}

// Stream is an in-process Kafka record stream for a consumer group.
type Stream struct {
	client  *kgo.Client
	records chan Record

	cancel context.CancelFunc
	done   chan struct{}

	mu  sync.Mutex
	err error

	// pending holds delivered but not yet Ack'd records of the current fetch,
	// per partition in delivery order. Offsets need not be contiguous
	// (compaction, transaction markers), so the head is the only valid Ack.
	pending map[partitionKey][]*kgo.Record
	// delivering is true while the current fetch is still being sent on records.
	delivering bool
	// fetchDone is closed once the current fetch is fully delivered and Ack'd.
	fetchDone chan struct{}
}

// Consume opens a consumer group for cfg and starts delivering records.
// The returned [Stream] must be closed with [Stream.Close] when finished.
func Consume(ctx context.Context, cfg Config) (*Stream, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	opts, err := clientOpts(cfg)
	if err != nil {
		return nil, err
	}
	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("kafka: create client: %w", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	s := &Stream{
		client:  client,
		records: make(chan Record),
		cancel:  cancel,
		done:    make(chan struct{}),
		pending: make(map[partitionKey][]*kgo.Record),
	}

	go s.loop(runCtx)
	return s, nil
}

func clientOpts(cfg Config) ([]kgo.Opt, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ConsumerGroup(cfg.GroupID),
		kgo.ConsumeTopics(cfg.Topics...),
		kgo.DisableAutoCommit(),
		kgo.BlockRebalanceOnPoll(),
	}
	if cfg.ClientID != "" {
		opts = append(opts, kgo.ClientID(cfg.ClientID))
	}
	if cfg.TLS != nil {
		opts = append(opts, kgo.DialTLSConfig(cfg.TLS))
	}
	if cfg.SASL.Mechanism != "" {
		mech, err := saslMechanism(cfg.SASL)
		if err != nil {
			return nil, err
		}
		opts = append(opts, kgo.SASL(mech))
	}
	return opts, nil
}

func saslMechanism(cfg SASLConfig) (sasl.Mechanism, error) {
	user := cfg.Username
	pass := cfg.Password
	authFn := func(context.Context) (plain.Auth, error) {
		return plain.Auth{User: user, Pass: pass}, nil
	}
	scramFn := func(context.Context) (scram.Auth, error) {
		return scram.Auth{User: user, Pass: pass}, nil
	}
	switch cfg.Mechanism {
	case SASLPlain:
		return plain.Plain(authFn), nil
	case SASLScramSHA256:
		return scram.Sha256(scramFn), nil
	case SASLScramSHA512:
		return scram.Sha512(scramFn), nil
	default:
		return nil, fmt.Errorf("kafka: unsupported SASL mechanism %q", cfg.Mechanism)
	}
}

// Records returns the channel of delivered records. It is closed when the
// stream context is cancelled or the client fails. After close, call [Stream.Err].
// Consume it from a single goroutine and [Record.Ack] each record in order.
func (s *Stream) Records() <-chan Record {
	return s.records
}

// Err returns the failure that stopped the stream, if any.
// It is safe to call after [Stream.Records] has closed.
func (s *Stream) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// Close cancels the stream, waits for the poll loop to exit, and closes the client.
func (s *Stream) Close() {
	s.cancel()
	<-s.done
	s.client.Close()
}

func (s *Stream) setErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err == nil {
		s.err = err
	}
}

func (s *Stream) loop(ctx context.Context) {
	defer close(s.done)
	defer close(s.records)

	// Every return path must AllowRebalance: BlockRebalanceOnPoll otherwise
	// keeps the group stuck, and Client.Close would wait on it.
	defer s.client.AllowRebalance()

	for {
		fetches := s.client.PollFetches(ctx)
		if fetches.IsClientClosed() || ctx.Err() != nil {
			return
		}
		if err := fatalFetchError(fetches); err != nil {
			s.setErr(err)
			return
		}
		if fetches.NumRecords() == 0 {
			s.client.AllowRebalance()
			continue
		}

		done := s.deliverFetch(ctx, fetches)
		select {
		case <-done:
		case <-ctx.Done():
			return
		}
		s.client.AllowRebalance()
	}
}

// fatalFetchError returns the first fetch error the client will not recover from.
// Retriable broker errors and transport errors are retried internally by kgo.
func fatalFetchError(fetches kgo.Fetches) error {
	for _, fe := range fetches.Errors() {
		if errors.Is(fe.Err, context.Canceled) || errors.Is(fe.Err, context.DeadlineExceeded) {
			continue
		}
		var kerrErr *kerr.Error
		if errors.As(fe.Err, &kerrErr) && !kerrErr.Retriable {
			return fmt.Errorf("kafka: fetch %s/%d: %w", fe.Topic, fe.Partition, fe.Err)
		}
	}
	return nil
}

// deliverFetch sends every record of fetches on s.records and returns a channel
// that is closed once all of them have been Ack'd.
func (s *Stream) deliverFetch(ctx context.Context, fetches kgo.Fetches) <-chan struct{} {
	s.mu.Lock()
	s.pending = make(map[partitionKey][]*kgo.Record)
	s.delivering = true
	s.fetchDone = make(chan struct{})
	done := s.fetchDone
	s.mu.Unlock()

	fetches.EachPartition(func(p kgo.FetchTopicPartition) {
		for _, kr := range p.Records {
			if ctx.Err() != nil {
				return
			}
			pk := partitionKey{topic: kr.Topic, partition: kr.Partition}
			s.mu.Lock()
			s.pending[pk] = append(s.pending[pk], kr)
			s.mu.Unlock()

			select {
			case s.records <- recordFromKgo(s, kr):
			case <-ctx.Done():
				return
			}
		}
	})

	s.mu.Lock()
	s.delivering = false
	s.finishFetchLocked()
	s.mu.Unlock()
	return done
}

// finishFetchLocked closes fetchDone when the fetch is fully delivered and Ack'd.
func (s *Stream) finishFetchLocked() {
	if s.delivering || len(s.pending) > 0 {
		return
	}
	select {
	case <-s.fetchDone:
	default:
		close(s.fetchDone)
	}
}

func (s *Stream) ack(ctx context.Context, r Record) error {
	pk := partitionKey{topic: r.Topic, partition: r.Partition}

	s.mu.Lock()
	queue := s.pending[pk]
	if len(queue) == 0 || queue[0] != r.krec {
		s.mu.Unlock()
		if len(queue) == 0 {
			return fmt.Errorf("kafka: ack for %s/%d@%d: no outstanding record on this partition",
				r.Topic, r.Partition, r.Offset)
		}
		return fmt.Errorf("kafka: out-of-order ack for %s/%d: got offset %d, want %d",
			r.Topic, r.Partition, r.Offset, queue[0].Offset)
	}
	s.mu.Unlock()

	if err := s.client.CommitRecords(ctx, r.krec); err != nil {
		return fmt.Errorf("kafka: commit %s/%d@%d: %w", r.Topic, r.Partition, r.Offset, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if queue := s.pending[pk]; len(queue) > 0 && queue[0] == r.krec {
		if len(queue) == 1 {
			delete(s.pending, pk)
		} else {
			s.pending[pk] = queue[1:]
		}
	}
	s.finishFetchLocked()
	return nil
}

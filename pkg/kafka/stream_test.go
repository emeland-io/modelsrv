package kafka_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.emeland.io/modelsrv/pkg/kafka"
)

const testTopic = "landscape-events"

var _ = Describe("Consume stream", func() {
	var (
		cluster *kfake.Cluster
		brokers []string
	)

	BeforeEach(func() {
		var err error
		cluster, err = kfake.NewCluster(
			kfake.NumBrokers(1),
			kfake.SeedTopics(1, testTopic),
		)
		Expect(err).NotTo(HaveOccurred())
		brokers = cluster.ListenAddrs()
	})

	AfterEach(func() {
		if cluster != nil {
			cluster.Close()
		}
	})

	produceAll := func(recs ...*kgo.Record) {
		cl, err := kgo.NewClient(
			kgo.SeedBrokers(brokers...),
			kgo.DefaultProduceTopic(testTopic),
		)
		Expect(err).NotTo(HaveOccurred())
		defer cl.Close()
		Expect(cl.ProduceSync(context.Background(), recs...).FirstErr()).NotTo(HaveOccurred())
	}

	produce := func(key, value []byte) {
		produceAll(&kgo.Record{Key: key, Value: value})
	}

	consume := func(ctx context.Context, group string) *kafka.Stream {
		stream, err := kafka.Consume(ctx, kafka.Config{
			Brokers:  brokers,
			GroupID:  group,
			Topics:   []string{testTopic},
			ClientID: "test-sensor",
		})
		Expect(err).NotTo(HaveOccurred())
		return stream
	}

	readOne := func(stream *kafka.Stream, timeout time.Duration) kafka.Record {
		GinkgoHelper()
		select {
		case rec, ok := <-stream.Records():
			Expect(ok).To(BeTrue(), "records channel closed: %v", stream.Err())
			return rec
		case <-time.After(timeout):
			Fail("timed out waiting for record")
			return kafka.Record{}
		}
	}

	It("delivers a produced record with the same key, value, and topic", func() {
		produce([]byte("k1"), []byte(`{"type":"com.example.created"}`))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		stream := consume(ctx, "group-deliver")
		defer stream.Close()

		rec := readOne(stream, 10*time.Second)
		Expect(rec.Topic).To(Equal(testTopic))
		Expect(rec.Key).To(Equal([]byte("k1")))
		Expect(rec.Value).To(Equal([]byte(`{"type":"com.example.created"}`)))
		Expect(rec.Ack(ctx)).To(Succeed())
	})

	It("commits on Ack so a later consumer in the same group does not see the record", func() {
		produce([]byte("acked"), []byte("payload-1"))

		ctx1, cancel1 := context.WithCancel(context.Background())
		stream1 := consume(ctx1, "group-ack")
		rec := readOne(stream1, 10*time.Second)
		Expect(rec.Value).To(Equal([]byte("payload-1")))
		Expect(rec.Ack(ctx1)).To(Succeed())
		stream1.Close()
		cancel1()

		produce([]byte("second"), []byte("payload-2"))

		ctx2, cancel2 := context.WithCancel(context.Background())
		defer cancel2()
		stream2 := consume(ctx2, "group-ack")
		defer stream2.Close()

		rec2 := readOne(stream2, 10*time.Second)
		Expect(rec2.Value).To(Equal([]byte("payload-2")), "acked record must not be redelivered")
		Expect(rec2.Ack(ctx2)).To(Succeed())
	})

	It("redelivers when closed without Ack", func() {
		produce([]byte("unacked"), []byte("see-again"))

		ctx1, cancel1 := context.WithCancel(context.Background())
		stream1 := consume(ctx1, "group-no-ack")
		rec := readOne(stream1, 10*time.Second)
		Expect(rec.Value).To(Equal([]byte("see-again")))
		// no Ack
		stream1.Close()
		cancel1()

		ctx2, cancel2 := context.WithCancel(context.Background())
		defer cancel2()
		stream2 := consume(ctx2, "group-no-ack")
		defer stream2.Close()

		rec2 := readOne(stream2, 10*time.Second)
		Expect(rec2.Value).To(Equal([]byte("see-again")))
		Expect(rec2.Ack(ctx2)).To(Succeed())
	})

	It("closes Records when the context is cancelled", func() {
		ctx, cancel := context.WithCancel(context.Background())
		stream := consume(ctx, "group-cancel")
		defer stream.Close()

		cancel()

		Eventually(func() bool {
			_, ok := <-stream.Records()
			return !ok
		}, 10*time.Second, 50*time.Millisecond).Should(BeTrue())
	})

	It("rejects out-of-order Ack", func() {
		// One produce call puts both records in the same batch, so they arrive
		// in one fetch and the second is delivered before the first is Ack'd.
		produceAll(
			&kgo.Record{Key: []byte("a"), Value: []byte("one")},
			&kgo.Record{Key: []byte("b"), Value: []byte("two")},
		)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		stream := consume(ctx, "group-order")
		defer stream.Close()

		first := readOne(stream, 10*time.Second)
		second := readOne(stream, 10*time.Second)
		Expect(second.Offset).To(BeNumerically(">", first.Offset))

		err := second.Ack(ctx)
		Expect(err).To(MatchError(ContainSubstring("out-of-order ack")))

		Expect(first.Ack(ctx)).To(Succeed())
		Expect(second.Ack(ctx)).To(Succeed())
	})
})

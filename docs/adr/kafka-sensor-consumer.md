# ADR: Kafka stream for Sensors

## Status

Accepted

## Context

Sensors need to consume Kafka topics that carry landscape change events. Record values
are CloudEvents; landscape resources sit in the CloudEvents `data` section. Mapping those
events into the model is context-specific and belongs in each Sensor binary.

The file-sensor path already separates transport (`pkg/filesensor` Sources) from format
parsing (`pkg/ingress`). Kafka is shared by every Sensor that reads a broker, so the
connection and offset commit belong in the modelsrv library. Decoding CloudEvents and
writing to `model.Model` must stay out of that library.

Alternatives considered:

- **Library applies documents after a Sensor transformer** — forces every Kafka Sensor
  through `ingress.Document` and couples offset commit to model apply.
- **`modelsrv server` flag for Kafka** — the reference server is a file-sensor; Kafka
  Sensors are separate binaries that embed the library.
- **HTTP endpoint that re-streams Kafka** — adds another hop when Sensors already run
  in-process with modelsrv.

## Decision

### Connection and ack live in `pkg/kafka`

[`pkg/kafka`](../../pkg/kafka) opens a consumer group for the topics a Sensor chooses and
exposes an in-process record stream. The Sensor acknowledges each record after it has
used (or deliberately dropped) it; acknowledgement commits the offset.

- No import of `pkg/ingress` or `pkg/model`.
- Record `Value` is the raw body (the CloudEvent). The library does not parse it.
- Manual commits only. Offsets advance only on contiguous acks per partition.
- TLS and SASL (`PLAIN`, `SCRAM-SHA-256`, `SCRAM-SHA-512`) are configured by the Sensor.

### Sensors own CloudEvents and model updates

Each Sensor:

1. Calls `kafka.Consume` with brokers, group id, and topics.
2. Reads `Stream.Records()`.
3. Decodes the CloudEvent and reads landscape resources from `data`.
4. Updates the model in its own way.
5. Calls `Record.Ack` when the event has been handled.

A record left unacknowledged is delivered again after restart. There is no dead-letter
topic and no retry policy inside the library.

### Out of scope

- CloudEvents parsing helpers in modelsrv.
- A `modelsrv server` Kafka flag.

## Consequences

- External Kafka Sensors import `pkg/kafka` and implement their own decode/apply loop.
- Adding a broker auth mechanism is a modelsrv change; adding an event mapping is a
  Sensor change.
- Consumer group id is chosen by the Sensor: one group shares partitions across
  replicas; separate groups each receive every record.

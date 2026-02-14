package events

// TODO: this is not used currently. Fix when needed.

type EventSlice[T any] interface {
	Len() int
	Get(i int) T
	// Append(event T)
}

type eventSlice[T any] struct {
	events []T
	sink   EventSink
}

func NewEventSlice[T any](sink EventSink) EventSlice[T] {
	return &eventSlice[T]{
		events: make([]T, 0),
		sink:   sink,
	}
}

var _ EventSlice[any] = (*eventSlice[any])(nil)

func (e *eventSlice[T]) Len() int {
	return len(e.events)
}

func (e *eventSlice[T]) Get(i int) T {
	return e.events[i]
}

/*
func (e *eventSlice[T]) Append(value T) {
	e.events = append(e.events, value)
	e.sink.Receive(GenericResource, CreateOperation, uuid.Nil, value)
}
*/

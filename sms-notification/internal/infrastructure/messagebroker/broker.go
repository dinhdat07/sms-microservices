package messagebroker

import "context"

type Message struct {
	ID     string
	Values map[string]interface{}
}

type MessageHandler func(ctx context.Context, msg Message) error

type Publisher interface {
	Publish(ctx context.Context, stream string, values []interface{}) error
}

type Subscriber interface {
	Subscribe(ctx context.Context, stream string, group string, consumer string, handler MessageHandler) error
}

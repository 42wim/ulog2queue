package main

import (
	"github.com/streadway/amqp"
)

type rabbitBackend struct {
	bulkRequest []*[]byte
	ctx         *Context
	queue       string
}

func NewRabbitConn(ctx *Context) (*amqp.Connection, *amqp.Channel, amqp.Queue) {
	conn, err := amqp.Dial(ctx.cfg.Backend["rabbit"].URI[0])
	failOnError(err, "Failed to connect to RabbitMQ")
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")

	q, err := ch.QueueDeclare(
		flagQueue, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		log.Fatal(err, "Failed to declare a queue")
	}
	return conn, ch, q
}

func NewRabbitBackend(ctx *Context, queue string) *rabbitBackend {
	return &rabbitBackend{ctx: ctx, queue: queue}
}

func (b *rabbitBackend) BulkAdd(line *[]byte) {
	b.bulkRequest = append(b.bulkRequest, line)
}

func (b *rabbitBackend) Close() {
	return
}

func (b *rabbitBackend) Flush() error {
	conn, ch, q := NewRabbitConn(b.ctx)
	defer conn.Close()
	defer ch.Close()
	log.Info("started rabbit task, messages in queue:", q.Messages)
	for _, line := range b.bulkRequest {
		err := ch.Publish(
			"",      // exchange
			b.queue, // routing key
			false,   // mandatory
			false,   // immediate
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        *line,
			})
		if err != nil {
			return err
		}
	}
	b.bulkRequest = b.bulkRequest[:0]
	return nil
}

func (b *rabbitBackend) NumberOfActions() int {
	return len(b.bulkRequest)
}

func (b *rabbitBackend) Ping() error {
	//TODO
	return nil
}

func (b *rabbitBackend) Restore() {
	//TODO
}

func (b *rabbitBackend) Size() int64 {
	conn, ch, q := NewRabbitConn(b.ctx)
	defer conn.Close()
	defer ch.Close()
	return int64(q.Messages)
}

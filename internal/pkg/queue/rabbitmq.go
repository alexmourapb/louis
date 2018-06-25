package queue

import (
	"github.com/streadway/amqp"
)

// DeclareQueue - declares queue ready to use by consumer and producer
func DeclareQueue(name string, ch *amqp.Channel) (amqp.Queue, error) {
	return ch.QueueDeclare(
		name,  // name
		true,  // durable [saved to file]
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
}

// SetQos tells RabbitMQ not to give more than one message
// to a worker at a time. Or, in other words, don't dispatch
// a new message to a worker until it has processed and acknowledged
// the previous one. Instead, it will dispatch it to the next worker that is not still busy.
func SetQos(ch *amqp.Channel) error {
	return ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
}

// Consume - returns iterator on messages
func Consume(ch *amqp.Channel, name string) (<-chan amqp.Delivery, error) {
	return ch.Consume(
		name,  // queue
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
}

// Publish - puts message int queue
func Publish(ch *amqp.Channel, exchangeName string, body []byte) error {
	return ch.Publish(
		exchangeName, // exchange
		"",           // routing key
		true,         // mandatory
		false,        // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         body,
		})
}

// DelcareExchange - declares exchange for putting messages
func DelcareExchange(ch *amqp.Channel, name string) error {
	return ch.ExchangeDeclare(
		name,     // name
		"fanout", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
}

func DeclareExchangeWithAE(ch *amqp.Channel, name, AEName string) error {
	var args = make(map[string]interface{})
	args["alternate-exchange"] = AEName
	return ch.ExchangeDeclare(
		name,     // name
		"fanout", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		args,     // arguments
	)

}

func BindQueueAndExchang(ch *amqp.Channel, queueName, exchangeName string) error {
	return ch.QueueBind(
		queueName,    // queue name
		"#",          // routing key
		exchangeName, // exchange
		false,        // nowait
		nil)
}

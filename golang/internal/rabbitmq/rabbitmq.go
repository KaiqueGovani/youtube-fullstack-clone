package rabbitmq

import (
	"fmt"

	"github.com/streadway/amqp"
)

type RabbitClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	url     string
}

func newConnection(url string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open a channel: %v", err)
	}

	return conn, channel, nil
}

func NewRabbitClient(connectionURL string) (*RabbitClient, error) {
	conn, channel, err := newConnection(connectionURL)
	if err != nil {
		return nil, err
	}
	return &RabbitClient{conn: conn, channel: channel, url: connectionURL}, nil
}

func (c *RabbitClient) ConsumeMessages(exchange, routingKey, queueName string) (<-chan amqp.Delivery, error) {
	err := c.channel.ExchangeDeclare(
		exchange,
		"direct",
		true,
		true,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare exchange: %v", err)
	}

	queue, err := c.channel.QueueDeclare(
		queueName,
		true,
		true,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %v", err)
	}

	err = c.channel.QueueBind(
		queue.Name,
		routingKey,
		exchange,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to bind queue: %v", err)
	}

	msgs, err := c.channel.Consume(
		queueName,
		"goapp",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to consume messages from queue %s: %v", queueName, err)
	}

	return msgs, nil
}

func (client *RabbitClient) Close() {
	client.channel.Close()
	client.conn.Close()
}
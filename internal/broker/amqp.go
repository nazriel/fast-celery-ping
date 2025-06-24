package broker

import (
	"context"
	"fmt"
	"time"

	"fast-celery-ping/internal/protocol"

	amqp "github.com/rabbitmq/amqp091-go"
)

// AMQPBroker implements the Broker interface for AMQP/RabbitMQ
type AMQPBroker struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	config     Config
	handler    *protocol.Handler
}

// NewAMQPBroker creates a new AMQP broker instance
func NewAMQPBroker(config Config) *AMQPBroker {
	return &AMQPBroker{
		config:  config,
		handler: protocol.NewHandler(),
	}
}

// Connect establishes connection to AMQP broker
func (a *AMQPBroker) Connect(ctx context.Context) error {
	var err error

	// Create connection with authentication if provided
	a.connection, err = amqp.Dial(a.config.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to AMQP broker: %w", err)
	}

	// Create channel
	a.channel, err = a.connection.Channel()
	if err != nil {
		a.connection.Close()
		return fmt.Errorf("failed to create AMQP channel: %w", err)
	}

	// Declare required exchanges
	err = a.declareExchanges()
	if err != nil {
		a.Close()
		return fmt.Errorf("failed to declare exchanges: %w", err)
	}

	// Test connection
	return a.Health(ctx)
}

// Close closes the AMQP connection and channel
func (a *AMQPBroker) Close() error {
	if a.channel != nil {
		a.channel.Close()
	}
	if a.connection != nil {
		return a.connection.Close()
	}
	return nil
}

// Health checks AMQP connectivity
func (a *AMQPBroker) Health(ctx context.Context) error {
	if a.connection == nil {
		return fmt.Errorf("AMQP connection not initialized")
	}

	if a.connection.IsClosed() {
		return fmt.Errorf("AMQP connection is closed")
	}

	if a.channel == nil {
		return fmt.Errorf("AMQP channel not initialized")
	}

	return nil
}

// declareExchanges declares the required AMQP exchanges for Celery
func (a *AMQPBroker) declareExchanges() error {
	// Declare the pidbox exchange (fanout exchange for broadcasting control messages)
	// Use passive declaration first to check if exchange exists with different type
	err := a.channel.ExchangeDeclarePassive(
		"celery.pidbox", // name
		"fanout",        // type
		true,            // durable
		false,           // auto-delete
		false,           // internal
		false,           // no-wait
		nil,             // args
	)
	if err != nil {
		// If passive declaration fails, try to declare the exchange
		err = a.channel.ExchangeDeclare(
			"celery.pidbox", // name
			"fanout",        // type
			true,            // durable
			false,           // auto-delete
			false,           // internal
			false,           // no-wait
			nil,             // args
		)
		if err != nil {
			return fmt.Errorf("failed to declare celery.pidbox exchange: %w", err)
		}
	}

	// Declare the reply exchange (direct exchange for reply messages)
	err = a.channel.ExchangeDeclarePassive(
		"reply.celery.pidbox", // name
		"direct",              // type
		true,                  // durable
		false,                 // auto-delete
		false,                 // internal
		false,                 // no-wait
		nil,                   // args
	)
	if err != nil {
		// If passive declaration fails, try to declare the exchange
		err = a.channel.ExchangeDeclare(
			"reply.celery.pidbox", // name
			"direct",              // type
			true,                  // durable
			false,                 // auto-delete
			false,                 // internal
			false,                 // no-wait
			nil,                   // args
		)
		if err != nil {
			return fmt.Errorf("failed to declare reply.celery.pidbox exchange: %w", err)
		}
	}

	return nil
}

// Ping implements the Celery ping functionality for AMQP
func (a *AMQPBroker) Ping(ctx context.Context, timeout time.Duration, destinations []string) (map[string]PingResponse, error) {
	if a.connection == nil || a.channel == nil {
		return nil, fmt.Errorf("AMQP connection not initialized")
	}

	// Create reply queue with simple UUID format
	replyTo := a.handler.CreateReplyQueue()

	// Declare temporary reply queue
	replyQueue, err := a.channel.QueueDeclare(
		replyTo, // name
		false,   // durable
		true,    // delete when unused
		true,    // exclusive
		false,   // no-wait
		nil,     // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare reply queue: %w", err)
	}

	// Bind reply queue to reply exchange
	err = a.channel.QueueBind(
		replyQueue.Name,       // queue name
		replyTo,               // routing key
		"reply.celery.pidbox", // exchange
		false,                 // no-wait
		nil,                   // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to bind reply queue: %w", err)
	}

	// Create ping message in raw format (direct JSON control message)
	pingData, err := a.handler.CreatePingMessage(replyTo, destinations, protocol.MessageFormatRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to create ping message: %w", err)
	}

	// Publish the ping message to the broadcast exchange
	err = a.channel.PublishWithContext(
		ctx,
		"celery.pidbox", // exchange
		"",              // routing key (empty for broadcast)
		false,           // mandatory
		false,           // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         pingData,
			DeliveryMode: amqp.Persistent,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to publish ping message: %w", err)
	}

	// Consume responses from reply queue
	responses := make(map[string]PingResponse)
	msgs, err := a.channel.Consume(
		replyQueue.Name, // queue
		"",              // consumer
		true,            // auto-ack
		false,           // exclusive
		false,           // no-local
		false,           // no-wait
		nil,             // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start consuming replies: %w", err)
	}

	// Wait for responses with timeout
	deadline := time.After(timeout)
	responseTimeout := time.NewTimer(100 * time.Millisecond) // Small timeout between responses

	for {
		select {
		case <-ctx.Done():
			return responses, ctx.Err()

		case <-deadline:
			// Timeout reached, return collected responses
			return responses, nil

		case msg, ok := <-msgs:
			if !ok {
				// Channel closed
				return responses, nil
			}

			// Reset response timeout for next message
			responseTimeout.Reset(100 * time.Millisecond)

			// Process the response
			response, err := a.handler.ParseWorkerResponse(msg.Body)
			if err != nil {
				continue
			}

			if a.handler.ValidateResponse(response) {
				workerName := a.handler.ExtractWorkerName(response)
				if workerName != "" {
					// Add response (map will naturally deduplicate)
					responses[workerName] = PingResponse{
						WorkerName: workerName,
						Status:     "pong",
						Timestamp:  time.Now().Unix(),
					}
				}
			}

		case <-responseTimeout.C:
			// Small timeout between responses to avoid waiting too long
			// if no more responses are coming
			if len(responses) > 0 {
				return responses, nil
			}
		}
	}
}

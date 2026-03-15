package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	reconnectDelay = 5 * time.Second
	maxRetries     = 5
	maxBodyBytes   = 64 * 1024 // 64 KB — guard against oversized messages

	dlxExchange = "notification.dlx"
)

// Consumer manages RabbitMQ connections and starts event consumers.
type Consumer struct {
	rawURL  string
	conn    *amqp.Connection

	otpHandler        OTPHandler
	customerHandler   CustomerHandler
	checkoutHandler   CheckoutHandler
	schedulingHandler SchedulingHandler
}

// OTPHandler handles OTP-related events from auth.events exchange.
type OTPHandler interface {
	HandleOTPRequested(ctx context.Context, event OTPRequestedEvent) error
}

// CustomerHandler handles customer lifecycle events from customer.events exchange.
type CustomerHandler interface {
	HandleCustomerCreated(ctx context.Context, event CustomerCreatedEvent) error
}

// CheckoutHandler handles payment and subscription events from checkout.events exchange.
type CheckoutHandler interface {
	HandlePaymentConfirmed(ctx context.Context, event PaymentConfirmedEvent) error
	HandleSubscriptionActivated(ctx context.Context, event SubscriptionActivatedEvent) error
}

// SchedulingHandler handles scheduling reminder events from scheduling.events exchange.
type SchedulingHandler interface {
	HandleAppointmentReminder(ctx context.Context, event AppointmentReminderEvent) error
}

func New(
	rawURL string,
	otpHandler OTPHandler,
	customerHandler CustomerHandler,
	checkoutHandler CheckoutHandler,
	schedulingHandler SchedulingHandler,
) *Consumer {
	return &Consumer{
		rawURL:            rawURL,
		otpHandler:        otpHandler,
		customerHandler:   customerHandler,
		checkoutHandler:   checkoutHandler,
		schedulingHandler: schedulingHandler,
	}
}

func sanitizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "<invalid-url>"
	}
	if u.User != nil {
		u.User = url.User("***")
	}
	return u.String()
}

// Start connects to RabbitMQ and begins consuming from all exchanges.
// Blocks until ctx is cancelled. Automatically reconnects on connection failure.
func (c *Consumer) Start(ctx context.Context) {
	for {
		if err := c.connect(); err != nil {
			log.Printf("[consumer] failed to connect (url: %s): %v — retrying in %s",
				sanitizeURL(c.rawURL), err, reconnectDelay)
			select {
			case <-ctx.Done():
				return
			case <-time.After(reconnectDelay):
				continue
			}
		}

		log.Println("[consumer] ✅ Connected to RabbitMQ")
		err := c.startAll(ctx)
		c.close()

		if ctx.Err() != nil {
			log.Println("[consumer] context cancelled — shutting down")
			return
		}
		log.Printf("[consumer] connection lost: %v — reconnecting in %s", err, reconnectDelay)
		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectDelay):
		}
	}
}

func (c *Consumer) connect() error {
	var err error
	for i := 0; i < maxRetries; i++ {
		c.conn, err = amqp.Dial(c.rawURL)
		if err == nil {
			return nil
		}
		log.Printf("[consumer] connection attempt %d/%d failed: %v", i+1, maxRetries, err)
		time.Sleep(time.Duration(i+1) * 2 * time.Second)
	}
	return fmt.Errorf("failed to connect after %d retries: %w", maxRetries, err)
}

func (c *Consumer) close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

type binding struct {
	exchange   string
	routingKey string
	queue      string
	handler    func(ctx context.Context, body []byte) error
}

// startAll declares all exchanges/queues and starts a goroutine per binding.
// Each binding gets its own AMQP channel (amqp.Channel is not goroutine-safe).
// Returns when any consumer exits or ctx is cancelled.
func (c *Consumer) startAll(ctx context.Context) error {
	bindings := []binding{
		{
			exchange:   "auth.events",
			routingKey: "otp.requested",
			queue:      "notification.auth.otp.requested",
			handler:    c.handleOTPRequested,
		},
		{
			exchange:   "customer.events",
			routingKey: "customer.created",
			queue:      "notification.customer.created",
			handler:    c.handleCustomerCreated,
		},
		{
			exchange:   "checkout.events",
			routingKey: "payment.confirmed",
			queue:      "notification.checkout.payment.confirmed",
			handler:    c.handlePaymentConfirmed,
		},
		{
			exchange:   "checkout.events",
			routingKey: "subscription.activated",
			queue:      "notification.checkout.subscription.activated",
			handler:    c.handleSubscriptionActivated,
		},
		{
			exchange:   "scheduling.events",
			routingKey: "reminder.24h",
			queue:      "notification.scheduling.reminder.24h",
			handler:    c.handleAppointmentReminder,
		},
	}

	// Declare dead-letter exchange once.
	setupCh, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open setup channel: %w", err)
	}
	if err := setupCh.ExchangeDeclare(
		dlxExchange, "topic", true, false, false, false, nil,
	); err != nil {
		setupCh.Close()
		return fmt.Errorf("failed to declare DLX: %w", err)
	}
	setupCh.Close()

	// Each binding gets its own channel (amqp.Channel is not goroutine-safe).
	type consumerCh struct {
		ch      *amqp.Channel
		binding binding
	}

	consumers := make([]consumerCh, 0, len(bindings))
	for _, b := range bindings {
		ch, err := c.conn.Channel()
		if err != nil {
			for _, cc := range consumers {
				cc.ch.Close()
			}
			return fmt.Errorf("failed to open channel for %s: %w", b.queue, err)
		}
		if err := c.declareAndBind(ch, b); err != nil {
			ch.Close()
			for _, cc := range consumers {
				cc.ch.Close()
			}
			return fmt.Errorf("setup failed for %s: %w", b.queue, err)
		}
		consumers = append(consumers, consumerCh{ch: ch, binding: b})
	}

	// Start one goroutine per binding; collect first error.
	errc := make(chan error, len(consumers))
	var wg sync.WaitGroup
	for _, cc := range consumers {
		cc := cc
		wg.Add(1)
		go func() {
			defer wg.Done()
			errc <- c.consume(ctx, cc.ch, cc.binding.queue, cc.binding.handler)
		}()
	}

	// Close all channels when done (regardless of exit path).
	go func() {
		wg.Wait()
		for _, cc := range consumers {
			cc.ch.Close()
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errc:
		return err
	}
}

func (c *Consumer) declareAndBind(ch *amqp.Channel, b binding) error {
	if err := ch.ExchangeDeclare(
		b.exchange, "topic", true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("exchange declare %s: %w", b.exchange, err)
	}

	args := amqp.Table{
		"x-dead-letter-exchange":    dlxExchange,
		"x-dead-letter-routing-key": b.routingKey + ".failed",
	}
	if _, err := ch.QueueDeclare(
		b.queue, true, false, false, false, args,
	); err != nil {
		return fmt.Errorf("queue declare %s: %w", b.queue, err)
	}

	if err := ch.QueueBind(b.queue, b.routingKey, b.exchange, false, nil); err != nil {
		return fmt.Errorf("queue bind %s: %w", b.queue, err)
	}
	return nil
}

// consume reads messages from a queue and calls handler for each one.
func (c *Consumer) consume(ctx context.Context, ch *amqp.Channel, queue string, handler func(context.Context, []byte) error) error {
	msgs, err := ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume %s: %w", queue, err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("channel closed for queue %s", queue)
			}

			// Guard against oversized messages.
			if len(msg.Body) > maxBodyBytes {
				log.Printf("[consumer] oversized message on %s (%d bytes) — discarding", queue, len(msg.Body))
				_ = msg.Nack(false, false)
				continue
			}

			if err := handler(ctx, msg.Body); err != nil {
				log.Printf("[consumer] handler error for queue %s (msgId=%s): %v — sending to DLX", queue, msg.MessageId, err)
				_ = msg.Nack(false, false)
			} else {
				_ = msg.Ack(false)
			}
		}
	}
}

// Dispatch methods — decode JSON and delegate to typed handlers.

func (c *Consumer) handleOTPRequested(ctx context.Context, body []byte) error {
	var event OTPRequestedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("decode OTPRequestedEvent: %w", err)
	}
	return c.otpHandler.HandleOTPRequested(ctx, event)
}

func (c *Consumer) handleCustomerCreated(ctx context.Context, body []byte) error {
	var event CustomerCreatedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("decode CustomerCreatedEvent: %w", err)
	}
	return c.customerHandler.HandleCustomerCreated(ctx, event)
}

func (c *Consumer) handlePaymentConfirmed(ctx context.Context, body []byte) error {
	var event PaymentConfirmedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("decode PaymentConfirmedEvent: %w", err)
	}
	return c.checkoutHandler.HandlePaymentConfirmed(ctx, event)
}

func (c *Consumer) handleSubscriptionActivated(ctx context.Context, body []byte) error {
	var event SubscriptionActivatedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("decode SubscriptionActivatedEvent: %w", err)
	}
	return c.checkoutHandler.HandleSubscriptionActivated(ctx, event)
}

func (c *Consumer) handleAppointmentReminder(ctx context.Context, body []byte) error {
	var event AppointmentReminderEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("decode AppointmentReminderEvent: %w", err)
	}
	return c.schedulingHandler.HandleAppointmentReminder(ctx, event)
}

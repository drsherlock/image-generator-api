package main

import (
	"encoding/json"
	"github.com/streadway/amqp"
	"log"
)

type queue struct {
	conn *amqp.Connection
	q    amqp.Queue
	ch   *amqp.Channel
	ex   string
}

func NewQueue(queueAddr string) *queue {
	conn, err := amqp.Dial(queueAddr)
	if err != nil {
		log.Fatalf("%s", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("%s", err)
	}

	args := make(amqp.Table)
	args["x-delayed-type"] = "direct"
	if err := ch.ExchangeDeclare("delayed", "x-delayed-message", true, false, false, false, args); err != nil {
		log.Fatalf("%s", err)
	}

	q, err := ch.QueueDeclare("delete-files", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("%s", err)
	}
	err = ch.QueueBind(q.Name, "delete.files", "delayed", false, nil)
	if err != nil {
		log.Fatalf("%s", err)
	}

	return &queue{
		conn: conn,
		q:    q,
		ch:   ch,
		ex:   "delayed",
	}
}

func (queue *queue) Publish(imageName ImageName) error {
	body, err := json.Marshal(imageName)
	if err != nil {
		return err
	}

	headers := make(amqp.Table)
	headers["x-delay"] = 20000
	err = queue.ch.Publish(queue.ex, "delete.files", false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         body,
		Headers:      headers,
	})
	if err != nil {
		return err
	}
	return nil
}

func (queue *queue) Consume(op func(imageName *ImageName) error) {
	if err := queue.ch.Qos(1, 0, false); err != nil {
		log.Printf("Could not configure QoS : %s", err)
	}

	messageChannel, err := queue.ch.Consume(queue.q.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Printf("Could not register consumer: %s", err)
	}

	stopChan := make(chan bool)

	go func() {
		log.Printf("Consumer ready")
		for d := range messageChannel {
			log.Printf("Received a message: %s", d.Body)

			imageName := &ImageName{}

			err := json.Unmarshal(d.Body, imageName)
			if err != nil {
				log.Printf("Error decoding JSON: %s", err)
			}

			if err := op(imageName); err != nil {
				log.Printf("Error during operation : %s", err)
			}

			if err := d.Ack(false); err != nil {
				log.Printf("Error acknowledging message : %s", err)
			} else {
				log.Printf("Acknowledged message")
			}

		}
	}()

	// Stop for program termination
	<-stopChan
}

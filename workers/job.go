package workers

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/nerdyworm/sess/queue"
	"github.com/nerdyworm/sess/util"
	"github.com/streadway/amqp"
)

type Job struct {
	Name     string `json:"name"`
	Payload  []byte `json:"payload"`
	Tries    int
	Delivery amqp.Delivery `json:"-"`
	errors   []error
	tryAgain bool
}

func (job *Job) ReplyTimeoutOrDefault() time.Duration {
	return time.Second * 90
}

func (job *Job) Retry() {
	job.tryAgain = true
}

func (job *Job) ShouldRetry() bool {
	return job.tryAgain && job.Tries < 2
}

func (job *Job) IncrementTries() {
	job.Tries = job.Tries + 1
}

func (job *Job) AddError(err error) {
	if job.errors == nil {
		job.errors = make([]error, 0)
	}

	job.errors = append(job.errors, err)
}

func (job *Job) Ack() error {
	return job.Delivery.Ack(false)
}

func (job *Job) PublishAndWait() error {
	body, err := json.Marshal(job)
	if err != nil {
		log.Printf("Error Marshaling Job: %s\n", err)
		return err
	}

	channel, err := queue.Connection.Channel()
	if err != nil {
		log.Printf("Error Connection.Channel(): %s\n", err)
		return err
	}

	q, err := channel.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when usused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)

	if err != nil {
		log.Printf("Error QueueDeclare: %s\n", err)
		return err
	}

	msgs, err := channel.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Printf("Error Consume: %s\n", err)
		return err
	}

	corrId := util.RandomString(32)

	err = channel.Publish(
		"",
		TASK_QUEUE_NAME,
		false,
		false,
		amqp.Publishing{
			Body:          body,
			ContentType:   "application/json",
			CorrelationId: corrId,
			DeliveryMode:  amqp.Persistent,
			ReplyTo:       q.Name,
		},
	)

	if err != nil {
		log.Println(err)
		return err
	}

	for {
		select {
		case <-time.After(job.ReplyTimeoutOrDefault()):
			log.Println("Timedout Waiting for %s\n", corrId)
			return errors.New("Timeout")

		case delivery := <-msgs:
			if corrId == delivery.CorrelationId {
				return nil
			}
		}
	}

	return nil
}

func (j *Job) SendReply() error {
	return queue.Channel.Publish(
		"",                 // exchange
		j.Delivery.ReplyTo, // routing key
		false,              // mandatory
		false,              // immediate
		amqp.Publishing{
			CorrelationId: j.Delivery.CorrelationId,
		})

}

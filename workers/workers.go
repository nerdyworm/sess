package workers

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nerdyworm/sess/conversions"
	"github.com/nerdyworm/sess/queue"
	"github.com/streadway/amqp"
)

var (
	workers map[string]Worker
)

type Worker func(*Job, amqp.Delivery)

func Register(name string, fn Worker) {
	workers[name] = fn
}

type ConversionWorker struct {
	InstanceID string
	Options    conversions.Options
}

func init() {
	workers = make(map[string]Worker)
}

var (
	TASK_QUEUE_NAME = "task_queue"
	EXCHANGE_NAME   = "sess.workers"
)

func Run() {
	setupQueue()
	runWorkers()
}

func setupQueue() {
	err := queue.Channel.ExchangeDeclare(
		EXCHANGE_NAME, // name
		"direct",      // type
		true,          // durable
		false,         // auto-deleted
		false,         // internal
		false,         // no-wait
		nil,           // arguments
	)
	failOnError(err, "Failed to declare Exchange")

	_, err = queue.Channel.QueueDeclare(
		TASK_QUEUE_NAME, // name
		true,            // durable
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		nil,             // arguments
	)
	failOnError(err, "Failed to declare a queue")

	err = queue.Channel.QueueBind(
		TASK_QUEUE_NAME, // name
		"",              // key
		EXCHANGE_NAME,   // exchange
		false,           // no-wait
		nil,             // args
	)
	failOnError(err, "Failed to bind the quue")

	err = queue.Channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")
}

func runWorkers() {
	forever := make(chan bool)

	for i := 0; i < 40; i++ {
		go work(i)
	}

	log.Printf(" [x] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func work(n int) {
	msgs, err := queue.Channel.Consume(
		TASK_QUEUE_NAME, // queue
		"",              // consumer
		false,           // auto-ack
		false,           // exclusive
		false,           // no-local
		false,           // no-wait
		nil,             // args
	)
	failOnError(err, "Failed to register a consumer")

	for d := range msgs {
		start := time.Now()
		job := Job{Delivery: d}
		err := json.Unmarshal(d.Body, &job)
		if err != nil {
			log.Printf("Error Unmarshaling (%v) `%s`\n", err, string(d.Body))
			continue
		}

		log.Printf("[%d] Got %s %s", n, job.Name, job.Payload)
		if fn, ok := workers[job.Name]; ok {
			fn(&job, d)
			log.Printf("[%d] Finished %s %v", n, job.Name, time.Since(start))

			if job.ShouldRetry() {
				err = retryJob(&job, d)
				if err != nil {
					log.Printf("Error retyring %s %v \n", job.Name, err)
				}
			} else {
				if job.Tries > 1 {
					log.Printf("Max Retries %s %v\n", job.Name, string(job.Payload))
					d.Nack(false, false)
				}
				continue
			}
		} else {
			log.Printf("[%d] Could not find worker %s\n", n, job.Name)
			d.Nack(false, false)
		}

	}
}

func retryJob(job *Job, d amqp.Delivery) error {
	job.IncrementTries()

	body, _ := json.Marshal(job)
	log.Printf("Retrying job: %s", string(body))

	return queue.Channel.Publish(
		"",
		TASK_QUEUE_NAME,
		false,
		false,
		amqp.Publishing{
			Body:          body,
			ContentType:   d.ContentType,
			CorrelationId: d.CorrelationId,
			DeliveryMode:  d.DeliveryMode,
			ReplyTo:       d.ReplyTo,
		},
	)
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

package queue

import (
	"log"

	"github.com/streadway/amqp"
)

var (
	Connection *amqp.Connection
	Channel    *amqp.Channel
)

func Setup() {
	var err error

	Connection, err = amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}

	Channel, err = Connection.Channel()
	if err != nil {
		log.Fatal(err)
	}

}

func Shutdown() {
	defer Connection.Close()
	Channel.Close()
}

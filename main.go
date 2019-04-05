package main

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/streadway/amqp"
	"log"
	"net/http"
	"os"
	"time"
)

var AMQPConnection *amqp.Connection
var AMQPChannel *amqp.Channel

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func receiver(w http.ResponseWriter, r *http.Request) {
	callbackRequest := map[string]interface{}{}

	err := json.NewDecoder(r.Body).Decode(&callbackRequest)
	failOnError(err, "Can`t decode webHook callBack")

	callbackRequestJson, err := json.Marshal(callbackRequest)
	failOnError(err, "Can`t serialise webHook response")

	log.Printf("Received a callback: %s", callbackRequestJson)

	name := fmt.Sprintf("viber_incoming")

	err = AMQPChannel.Publish(
		"",
		name,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Transient,
			ContentType:  "application/json",
			Body:         callbackRequestJson,
			Timestamp:    time.Now(),
		})

	failOnError(err, "Failed to publish a message")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(callbackRequestJson)
}

func redirect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "./static/redirect.html")
}

func init() {
	err := godotenv.Load()
	failOnError(err, "Error loading .env file")

	cs := fmt.Sprintf("amqp://%s:%s@%s:%s/%s",
		os.Getenv("RABBITMQ_ERP_LOGIN"),
		os.Getenv("RABBITMQ_ERP_PASS"),
		os.Getenv("RABBITMQ_ERP_HOST"),
		os.Getenv("RABBITMQ_ERP_PORT"),
		os.Getenv("RABBITMQ_ERP_VHOST"))

	connection, err := amqp.Dial(cs)
	failOnError(err, "Failed to connect to RabbitMQ")
	AMQPConnection = connection
	//defer connection.Close()

	channel, err := AMQPConnection.Channel()
	failOnError(err, "Failed to open a channel")
	AMQPChannel = channel

	failOnError(err, "Failed to declare a queue")
}

func main() {
	http.HandleFunc("/", receiver)
	http.HandleFunc("/redirect", redirect)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("LISTEN_PORT")), nil))
	defer AMQPConnection.Close()
	defer AMQPChannel.Close()
}

package handlers

import (
	"Storm-Hunt/storm-backend/models"
	"encoding/json"
	"net/http"
	"os"

	"github.com/rs/zerolog/log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func SendWeatherTask(w http.ResponseWriter, r *http.Request) {
	// Парсим запрос (например, JSON: {"region":"Atlantic","user_id":"123"})
	var task models.WeatherTask
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Подключение к RabbitMQ
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Printf("Failed to connect to RabbitMQ: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Failed to open channel: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer ch.Close()

	// Декларируем очередь
	q, err := ch.QueueDeclare(
		"weather_tasks", // Имя
		true,            // Durable
		false,           // Auto-delete
		false,           // Exclusive
		false,           // No-wait
		nil,             // Args
	)
	if err != nil {
		log.Printf("Failed to declare queue: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Сериализуем задачу
	body, err := json.Marshal(task)
	if err != nil {
		log.Printf("Failed to marshal task: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Публикуем в очередь
	err = ch.Publish(
		"",     // Exchange
		q.Name, // Routing key
		false,  // Mandatory
		false,  // Immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		log.Printf("Failed to publish: %v", err)
		http.Error(w, "Failed to send task", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Task sent for region: " + task.Region})
}

package tasks

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
	var task models.StreamTask
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL")) // Подключение к RabbitMQ
	if err != nil {
		log.Printf("Failed to connect to RabbitMQ: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	ch, err := conn.Channel() // Открытие канала
	if err != nil {
		log.Printf("Failed to open channel: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer ch.Close()

	qname := "weather-tasks"
	dlxName := "dlx.weather"
	dlxRoutingKey := "failed.weather"

	if err := DeclareQueue(ch, qname, dlxName, dlxRoutingKey); err != nil {
		log.Printf("Failed to declare queue: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(task) // Сериализация задачи
	if err != nil {
		log.Printf("Failed to marshal task: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = ch.Publish( // Публикация в очередь
		"",
		qname,
		false,
		false,
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

	w.WriteHeader(http.StatusOK) // Возврат статуса 200 ОК при успехе
	json.NewEncoder(w).Encode(map[string]string{"message": "Task sent for region: " + task.Region})
}

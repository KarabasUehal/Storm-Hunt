package tasks

import (
	"Storm-Hunt/storm-backend/models"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

func SendWebcamsTask(c *gin.Context) {
	// Парсим запрос (например, JSON: {"region":"Atlantic","user_id":"123"})
	var task models.WebcamTask
	if err := c.ShouldBindJSON(&task); err != nil {
		log.Error().Err(err).Msg("Error binding json")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to bind json",
		})
		return
	}

	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Error().Err(err).Msg("Error connection to RabbitMQ")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to connect to RabbitMQ",
		})
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Error().Err(err).Msg("Error opening channel")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to open task chanel",
		})
		return
	}
	defer ch.Close()

	qname := "webcams-task"
	dlxName := "dlx.webcams"
	dlxRoutingKey := "failed.webcams"

	if err := DeclareQueue(ch, qname, dlxName, dlxRoutingKey); err != nil {
		log.Error().Err(err).Msg("Error to declare queue")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to declare queue",
		})
		return
	}

	// Сериализация задачи
	body, err := json.Marshal(task)
	if err != nil {
		log.Error().Err(err).Msg("Error serializing task")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to marhal task",
		})
		return
	}

	// Отправка задачи в очередь
	err = ch.Publish(
		"",    // Default exchange
		qname, // routing key (имя очереди)
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("Error publishing task")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to publish task",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Webcam task sent for camera: %s", task.CameraID),
	})
}

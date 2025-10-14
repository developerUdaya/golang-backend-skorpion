package messaging

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writers map[string]*kafka.Writer
}

type KafkaConsumer struct {
	readers map[string]*kafka.Reader
}

func NewKafkaProducer(brokers []string) *KafkaProducer {
	return &KafkaProducer{
		writers: make(map[string]*kafka.Writer),
	}
}

func NewKafkaConsumer(brokers []string, groupID string) *KafkaConsumer {
	return &KafkaConsumer{
		readers: make(map[string]*kafka.Reader),
	}
}

func (kp *KafkaProducer) GetWriter(topic string, brokers []string) *kafka.Writer {
	if writer, exists := kp.writers[topic]; exists {
		return writer
	}

	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	kp.writers[topic] = writer
	return writer
}

func (kp *KafkaProducer) SendMessage(topic string, brokers []string, key string, value interface{}) error {
	writer := kp.GetWriter(topic, brokers)

	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}

	message := kafka.Message{
		Key:   []byte(key),
		Value: jsonData,
	}

	return writer.WriteMessages(context.Background(), message)
}

func (kp *KafkaProducer) Close() {
	for _, writer := range kp.writers {
		writer.Close()
	}
}

func (kc *KafkaConsumer) GetReader(topic string, brokers []string, groupID string) *kafka.Reader {
	if reader, exists := kc.readers[topic]; exists {
		return reader
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	kc.readers[topic] = reader
	return reader
}

func (kc *KafkaConsumer) ConsumeMessages(topic string, brokers []string, groupID string, handler func([]byte) error) {
	reader := kc.GetReader(topic, brokers, groupID)

	for {
		message, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Error reading message from topic %s: %v", topic, err)
			continue
		}

		if err := handler(message.Value); err != nil {
			log.Printf("Error handling message: %v", err)
		}
	}
}

func (kc *KafkaConsumer) Close() {
	for _, reader := range kc.readers {
		reader.Close()
	}
}

// Event types for async processing
type OrderEvent struct {
	Type    string      `json:"type"`
	OrderID string      `json:"order_id"`
	UserID  string      `json:"user_id"`
	Data    interface{} `json:"data"`
}

type InventoryEvent struct {
	Type         string `json:"type"`
	ProductID    string `json:"product_id"`
	Quantity     int    `json:"quantity"`
	RestaurantID string `json:"restaurant_id"`
}

type NotificationEvent struct {
	Type     string                 `json:"type"`
	UserID   string                 `json:"user_id"`
	Title    string                 `json:"title"`
	Message  string                 `json:"message"`
	Metadata map[string]interface{} `json:"metadata"`
}

package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/gin-gonic/gin"
)

const (
	subscription    = ``
	vapidPublicKey  = "BLWWomxkvsI_TrblWIU8OjVKinMWPEEAOg2uS0XnbBnPJx2fJp16BL8Qy_8tPEzKQjUfs8ajScIM7pZ7aerLMqI"
	vapidPrivateKey = "ui_28cHIbyOWEcHHcc7iNS-o1g8aHdsz-RzSIM7zgI8"
)

var (
	subscriptions []webpush.Subscription
	mu            sync.Mutex // For thread-safe access to subscriptions
)

// type Sub struct {
// 	Subscription webpush.Subscription `json:"subscription"`
// }

// subscribeHandler handles new push subscriptions from the frontend
func SubscribeHandler(c *gin.Context) {
	var json webpush.Subscription
	err := c.ShouldBindJSON(&json)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	mu.Lock()
	subscriptions = append(subscriptions, json)
	mu.Unlock()

	log.Print(subscriptions)
	c.JSON(http.StatusOK, gin.H{"message": "Subscription Successful!"})
	return
}

func TestPush(c *gin.Context) {
	// Send Notification
	log.Printf("%+v\n", subscriptions)
	resp, err := webpush.SendNotification([]byte("Test"), &subscriptions[0], &webpush.Options{
		Subscriber:      "colinthatcher0@gmail.com", // Do not include "mailto:"
		VAPIDPublicKey:  vapidPublicKey,
		VAPIDPrivateKey: vapidPrivateKey,
		TTL:             30,
	})
	if err != nil {
		log.Fatal(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	c.JSON(http.StatusOK, gin.H{"message": "test notification sent"})
	return
}

func GenerateKeys(c *gin.Context) {
	secret, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success", "public_key": pub, "private_key": secret})
	return
}

// sendNotificationHandler allows triggering a notification (e.g., from an admin panel)
func SendNotificationHandler(c *gin.Context) {
	// In a real app, you'd get the message from the request body or a database
	message := "Hello from your Go backend!"
	title := "Go Push Notification"
	payload := map[string]string{
		"title": title,
		"body":  message,
		// "icon":  "/path/to/your/icon.png", // Optional: will be used by service worker
	}
	payloadBytes, _ := json.Marshal(payload) // Convert to JSON string

	mu.Lock()
	subsToSend := make([]webpush.Subscription, len(subscriptions))
	copy(subsToSend, subscriptions) // Copy to avoid holding lock during send
	mu.Unlock()

	if len(subsToSend) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No active subscriptions to send to."})
		return
	}

	for _, sub := range subsToSend {
		res, err := webpush.SendNotification(payloadBytes, &sub, &webpush.Options{
			Subscriber:      "colinthatcher0@gmail.com",
			TTL:             300, // Time-to-live in seconds
			VAPIDPrivateKey: vapidPrivateKey,
			VAPIDPublicKey:  vapidPublicKey,
			// Endpoint:        sub.Endpoint,
			// Auth:            sub.Keys["auth"],
			// P256DH:          sub.Keys["p256dh"],
		})
		if err != nil {
			log.Printf("Error sending notification to %s: %v", sub.Endpoint, err)
			// In a real app, you might want to remove invalid subscriptions here
			continue
		}
		res.Body.Close() // Important to close the response body
		log.Printf("Notification sent to %s (Status: %d)", sub.Endpoint, res.StatusCode)
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Notifications sent to %d subscribers!", len(subsToSend))})
}

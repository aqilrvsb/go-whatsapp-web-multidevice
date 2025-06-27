package config

import (
	"os"
	"strings"
)

func InitEnvironment() {
	// Override with environment variables if they exist
	if port := os.Getenv("APP_PORT"); port != "" {
		AppPort = port
	}
	
	if port := os.Getenv("PORT"); port != "" {
		// Railway provides PORT env var
		AppPort = port
	}
	
	if debug := os.Getenv("APP_DEBUG"); debug == "true" {
		AppDebug = true
	}
	
	if dbUri := os.Getenv("DB_URI"); dbUri != "" {
		DBURI = dbUri
	}
	
	if basicAuth := os.Getenv("APP_BASIC_AUTH"); basicAuth != "" {
		AppBasicAuthCredential = strings.Split(basicAuth, ",")
	}
	
	if autoReply := os.Getenv("WHATSAPP_AUTO_REPLY"); autoReply != "" {
		WhatsappAutoReplyMessage = autoReply
	}
	
	if webhook := os.Getenv("WHATSAPP_WEBHOOK"); webhook != "" {
		WhatsappWebhook = strings.Split(webhook, ",")
	}
	
	if webhookSecret := os.Getenv("WHATSAPP_WEBHOOK_SECRET"); webhookSecret != "" {
		WhatsappWebhookSecret = webhookSecret
	}
	
	if validation := os.Getenv("WHATSAPP_ACCOUNT_VALIDATION"); validation == "false" {
		WhatsappAccountValidation = false
	}
	
	if storage := os.Getenv("WHATSAPP_CHAT_STORAGE"); storage == "true" {
		WhatsappChatStorage = true
	} else {
		WhatsappChatStorage = false
	}
	
	// Redis settings
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		RedisURL = redisURL
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		RedisPassword = redisPassword
	}
	if redisHost := os.Getenv("REDISHOST"); redisHost != "" {
		RedisHost = redisHost
	}
	if redisPort := os.Getenv("REDISPORT"); redisPort != "" {
		RedisPort = redisPort
	}
}
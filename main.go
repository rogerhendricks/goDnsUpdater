package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	stateFile     = "last_ip.txt"
	checkInterval = 1 * time.Hour
)

var (
	telegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	telegramChatID   = os.Getenv("TELEGRAM_CHAT_ID")
)

// Telegram API response structures
type Update struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}

type TelegramUpdatesResponse struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

func main() {
	if telegramBotToken == "" || telegramChatID == "" {
		log.Fatal("Telegram credentials must be set in environment variables.")
	}

	log.Println("Starting interactive IP monitor service...")

	// 1. Start the hourly background check loop
	go startHourlyCheck()

	// 2. Start listening for incoming Telegram commands (Main loop)
	startBotListener()
}

// Background worker for the 1-hour proactive checks
func startHourlyCheck() {
	// Run an initial check on startup
	checkAndNotifyIfChanged()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		checkAndNotifyIfChanged()
	}
}

func checkAndNotifyIfChanged() {
	currentIP, err := getPublicIP()
	if err != nil {
		log.Printf("[Hourly] Error fetching IP: %v", err)
		return
	}

	lastIP := getLastKnownIP()

	if currentIP != lastIP {
		log.Printf("[Hourly] IP changed from %s to %s", lastIP, currentIP)
		msg := fmt.Sprintf("⚠️ <b>Homelab IP Changed</b>\n\nOld: `%s`\nNew: `%s`", lastIP, currentIP)
		
		if err := sendTelegramNotification(msg); err == nil {
			saveKnownIP(currentIP)
		}
	} else {
		log.Printf("[Hourly] IP remains unchanged: %s", currentIP)
	}
}

// Long polling loop to listen for your messages
func startBotListener() {
	offset := 0
	client := &http.Client{Timeout: 35 * time.Second} // Long polling timeout

	for {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=30", telegramBotToken, offset)
		resp, err := client.Get(url)
		if err != nil {
			log.Printf("[Listener] Error polling Telegram: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		var updateResp TelegramUpdatesResponse
		if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
			resp.Body.Close()
			log.Printf("[Listener] Error decoding JSON: %v", err)
			continue
		}
		resp.Body.Close()

		for _, update := range updateResp.Result {
			offset = update.UpdateID + 1

			if update.Message == nil {
				continue
			}

			// Security: Only respond if the message is from YOUR Chat ID
			strChatID := strconv.FormatInt(update.Message.Chat.ID, 10)
			if strChatID != telegramChatID {
				log.Printf("[Security] Ignored message from unauthorized Chat ID: %s", strChatID)
				continue
			}

			// Process commands
			command := strings.TrimSpace(strings.ToLower(update.Message.Text))
			if command == "/ip" || command == "/status" {
				log.Println("[Listener] Received manual IP request.")
				handleOnDemandRequest()
			}
		}
	}
}

func handleOnDemandRequest() {
	currentIP, err := getPublicIP()
	if err != nil {
		sendTelegramNotification("❌ Error: Could not fetch public IP address.")
		return
	}
	
	msg := fmt.Sprintf("ℹ️ <b>Current Homelab IP:</b>\n`%s`", currentIP)
	sendTelegramNotification(msg)
}

// Helper Functions
func getPublicIP() (string, error) {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ipBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(ipBytes)), nil
}

func getLastKnownIP() string {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return "Unknown"
	}
	return strings.TrimSpace(string(data))
}

func saveKnownIP(ip string) {
	_ = os.WriteFile(stateFile, []byte(ip), 0644)
}

func sendTelegramNotification(message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramBotToken)
	payload := map[string]string{
		"chat_id":    telegramChatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
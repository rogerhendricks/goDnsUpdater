# Telegram Homelab IP Monitor

A lightweight, secure, and containerized Go application designed for homelabs running on dynamic IPs. 

This service monitors your external IP address, notifies you via Telegram when it changes, and allows you to manually request your current IP on demand—all without needing to open any inbound ports on your router.

## Features
* **Proactive Monitoring:** Checks your public IP every hour and sends an alert if it changes.
* **On-Demand Checks:** Send `/ip` or `/status` to your bot at any time to instantly get your current IP.
* **Zero Inbound Ports:** Uses Telegram Long Polling, meaning your homelab reaches out to Telegram. No port forwarding required.
* **Ultra-Lightweight:** Compiled statically into an Alpine Linux container (typically < 15MB).

---

## Security Issues
**The Issue:** By default, if someone discovers your Telegram bot's username, they can send messages to it. If the bot blindly responded to `/ip` commands, it would leak your homelab's public IP address to strangers on the internet.

**The Solution:** Strict Chat ID Validation. 
The application evaluates the `chat_id` of every incoming message against your configured `TELEGRAM_CHAT_ID` environment variable. If a message arrives from an unauthorized user or group, the bot drops the request entirely and logs a security warning. 
```go
// Security check inside the listener:
strChatID := strconv.FormatInt(update.Message.Chat.ID, 10)
if strChatID != telegramChatID {
    log.Printf("[Security] Ignored message from unauthorized Chat ID: %s", strChatID)
    continue
}
```
## Deployment
### Prerequisites 

Docker and Docker Compose installed on your homelab.

A Telegram Bot Token (from @BotFather).

Your personal Telegram Chat ID (you can get this from bots like @userinfobot).

### Configuration (.env file)
In your project directory, create a hidden file named .env to securely store your credentials.

Note: Do not commit this file to version control.

Code snippet
```
TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrSTUvwxYZ
TELEGRAM_CHAT_ID=987654321
```
### Directory Structure
Ensure your project folder looks exactly like this before running Docker:

Plaintext
ip-monitor/
├── .env
├── docker-compose.yml
├── Dockerfile
└── main.go
### Docker Deployment
Create the data directory: This folder is mapped as a volume in your docker-compose.yml. It stores the last_ip.txt state file so your IP state persists across container restarts, preventing false notifications.

Bash
mkdir data

### Build and start the container: 
Run this command to build the Go binary and start the background service.
   ```bash
   docker compose up -d --build
   ```
Verify it is running: Check the logs to ensure the container started successfully and performed its initial IP check.

```bash
docker compose logs -f
```

---

## Usage
Once the container is running smoothly, open Telegram and send your bot the message:
`/ip` (or `/status`)

It will reply immediately with your homelab's current external IP address.
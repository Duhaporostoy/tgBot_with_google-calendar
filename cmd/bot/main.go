package main

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"tg-calendar-bot/internal/config"
	"tg-calendar-bot/internal/scheduler"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("🚀 Запуск бота...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка конфига: %v", err)
	}

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Fatalf("Ошибка Telegram: %v", err)
	}
	log.Printf("✅ Бот запущен: @%s", bot.Self.UserName)

	sched := scheduler.New(cfg, bot)
	sched.Run() // блокирует
}

package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken       string
	TelegramChatID      int64
	ICalURL             string
	MorningScheduleTime string
	ReminderMinutes     int
	Location            *time.Location
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	loc, err := time.LoadLocation(getEnv("TIMEZONE", "Europe/Moscow"))
	if err != nil {
		return nil, fmt.Errorf("неверный TIMEZONE: %w", err)
	}

	chatID, err := strconv.ParseInt(mustEnv("TELEGRAM_CHAT_ID"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("неверный TELEGRAM_CHAT_ID: %w", err)
	}

	reminderMin, err := strconv.Atoi(getEnv("REMINDER_MINUTES", "30"))
	if err != nil {
		return nil, fmt.Errorf("неверный REMINDER_MINUTES: %w", err)
	}

	return &Config{
		TelegramToken:       mustEnv("TELEGRAM_TOKEN"),
		TelegramChatID:      chatID,
		ICalURL:             mustEnv("ICAL_URL"),
		MorningScheduleTime: getEnv("MORNING_SCHEDULE_TIME", "09:00"),
		ReminderMinutes:     reminderMin,
		Location:            loc,
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Fprintf(os.Stderr, "ОШИБКА: переменная %s не задана в .env\n", key)
		os.Exit(1)
	}
	return v
}

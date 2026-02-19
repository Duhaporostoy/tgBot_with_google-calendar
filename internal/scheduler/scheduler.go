package scheduler

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"tg-calendar-bot/internal/bot"
	"tg-calendar-bot/internal/calendar"
	"tg-calendar-bot/internal/config"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Scheduler struct {
	cfg *config.Config
	tg  *tgbotapi.BotAPI

	mu          sync.Mutex
	reminded    map[string]bool
	knownEvents map[string]*calendar.Event
}

func New(cfg *config.Config, tg *tgbotapi.BotAPI) *Scheduler {
	return &Scheduler{
		cfg:         cfg,
		tg:          tg,
		reminded:    make(map[string]bool),
		knownEvents: make(map[string]*calendar.Event),
	}
}

func (s *Scheduler) Run() {
	log.Println("✅ Планировщик запущен")

	s.initKnownEvents()

	// Отправляем планы на неделю при старте
	go s.sendWeeklyOnStart()

	minuteTicker := time.NewTicker(1 * time.Minute)
	changeTicker := time.NewTicker(5 * time.Minute)
	for {
		select {
		case <-minuteTicker.C:
			go s.checkMorningSchedule()
			go s.checkReminders()
		case <-changeTicker.C:
			go s.checkChanges()
		}
	}
}

func (s *Scheduler) checkMorningSchedule() {
	now := time.Now().In(s.cfg.Location)
	if fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute()) != s.cfg.MorningScheduleTime {
		return
	}

	log.Println("📅 Отправляю утреннее расписание")

	events, err := calendar.TodayEvents(s.cfg.ICalURL, s.cfg.Location)
	if err != nil {
		log.Printf("Ошибка получения событий: %v", err)
		return
	}

	s.send(bot.FormatDaySchedule(events, s.cfg.Location))
}

func (s *Scheduler) checkReminders() {
	now := time.Now().In(s.cfg.Location)
	targetStart := now.Add(time.Duration(s.cfg.ReminderMinutes) * time.Minute)
	from := targetStart.Add(-30 * time.Second)
	to := targetStart.Add(30 * time.Second)

	events, err := calendar.FetchEvents(s.cfg.ICalURL, from, to, s.cfg.Location)
	if err != nil {
		log.Printf("Ошибка проверки напоминаний: %v", err)
		return
	}

	for _, ev := range events {
		key := fmt.Sprintf("%s-%d", ev.ID, s.cfg.ReminderMinutes)

		s.mu.Lock()
		already := s.reminded[key]
		if !already {
			s.reminded[key] = true
		}
		s.mu.Unlock()

		if already {
			continue
		}

		log.Printf("⏰ Напоминание: %s", ev.Title)
		s.send(bot.FormatReminder(ev, s.cfg.ReminderMinutes))
	}
}

func (s *Scheduler) checkChanges() {
	events, err := calendar.UpcomingEvents(s.cfg.ICalURL, s.cfg.Location)
	if err != nil {
		log.Printf("Ошибка проверки изменений: %v", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ev := range events {
		prev, exists := s.knownEvents[ev.ID]
		s.knownEvents[ev.ID] = ev

		if !exists {
			log.Printf("🆕 Новая встреча: %s", ev.Title)
			go s.send(bot.FormatEventChanged(ev, "new"))
			continue
		}

		if !prev.Start.Equal(ev.Start) {
			log.Printf("🔄 Перенос: %s", ev.Title)
			go s.send(bot.FormatEventChanged(ev, "moved"))
		}
	}

	for id, prev := range s.knownEvents {
		found := false
		for _, ev := range events {
			if ev.ID == id {
				found = true
				break
			}
		}
		if !found && prev.Start.After(time.Now()) {
			log.Printf("❌ Отменена: %s", prev.Title)
			go s.send(bot.FormatEventChanged(prev, "cancelled"))
			delete(s.knownEvents, id)
		}
	}
}

func (s *Scheduler) initKnownEvents() {
	events, err := calendar.UpcomingEvents(s.cfg.ICalURL, s.cfg.Location)
	if err != nil {
		log.Printf("Ошибка инициализации событий: %v", err)
		return
	}
	for _, ev := range events {
		s.knownEvents[ev.ID] = ev
	}
	log.Printf("📋 Загружено %d событий на ближайшую неделю", len(events))
}

func (s *Scheduler) send(text string) {
	msg := tgbotapi.NewMessage(s.cfg.TelegramChatID, text)
	msg.ParseMode = "MarkdownV2"
	msg.DisableWebPagePreview = true
	if _, err := s.tg.Send(msg); err != nil {
		log.Printf("Ошибка отправки: %v\nТекст: %s", err, text)
	}
}
func (s *Scheduler) sendWeeklyOnStart() {
	events, err := calendar.UpcomingEvents(s.cfg.ICalURL, s.cfg.Location)
	if err != nil {
		log.Printf("Ошибка получения событий на неделю: %v", err)
		return
	}

	now := time.Now().In(s.cfg.Location)
	var sb strings.Builder
	fmt.Fprintf(&sb, "🚀 *Бот запущен\\!*\n\n📆 *Планы на ближайшую неделю:*\n\n")

	if len(events) == 0 {
		sb.WriteString("_Встреч нет_ 🎉")
	} else {
		currentDay := ""
		for _, ev := range events {
			day := ev.Start.Format("02.01")
			if day != currentDay {
				currentDay = day
				weekday := russianWeekday(ev.Start.Weekday())
				sb.WriteString(fmt.Sprintf("📅 *%s, %s*\n", bot.EscMD(weekday), bot.EscMD(day)))
			}
			timeStr := ev.Start.Format("15:04")
			if !ev.End.IsZero() {
				timeStr += "–" + ev.End.Format("15:04")
			}
			sb.WriteString(fmt.Sprintf("  • %s \\(%s\\)\n", bot.EscMD(ev.Title), bot.EscMD(timeStr)))
			if ev.MeetLink != "" {
				sb.WriteString(fmt.Sprintf("    📹 [Meet](%s)\n", ev.MeetLink))
			}
		}
		sb.WriteString(fmt.Sprintf("\n_Всего встреч: %d_", len(events)))
	}
	_ = now
	s.send(sb.String())
}

func russianWeekday(w time.Weekday) string {
	days := []string{"Воскресенье", "Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота"}
	return days[w]
}

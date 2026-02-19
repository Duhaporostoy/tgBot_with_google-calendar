package bot

import (
	"fmt"
	"strings"
	"time"

	"tg-calendar-bot/internal/calendar"
)

func FormatDaySchedule(events []*calendar.Event, loc *time.Location) string {
	now := time.Now().In(loc)
	dateStr := russianDate(now)

	if len(events) == 0 {
		return fmt.Sprintf("📅 *%s*\n\nСвободный день, встреч нет 🎉", EscMD(dateStr))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📅 *%s* — расписание дня\n\n", EscMD(dateStr)))

	for i, ev := range events {
		sb.WriteString(formatShort(i+1, ev))
	}

	sb.WriteString(fmt.Sprintf("\n_Всего встреч: %d_", len(events)))
	return sb.String()
}

func FormatReminder(ev *calendar.Event, minutesBefore int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("⏰ *Через %d минут:* %s\n\n", minutesBefore, EscMD(ev.Title)))

	timeStr := ev.Start.Format("15:04")
	if !ev.End.IsZero() {
		timeStr += " – " + ev.End.Format("15:04")
	}
	sb.WriteString(fmt.Sprintf("🕐 %s\n", EscMD(timeStr)))

	if ev.Location != "" {
		sb.WriteString(fmt.Sprintf("📍 %s\n", EscMD(ev.Location)))
	}

	sb.WriteString(formatLinks(ev))

	if ev.Description != "" {
		desc := stripHTML(ev.Description)
		if len([]rune(desc)) > 300 {
			desc = string([]rune(desc)[:300]) + "…"
		}
		if desc != "" {
			sb.WriteString(fmt.Sprintf("\n_%s_\n", EscMD(desc)))
		}
	}

	return sb.String()
}

func FormatEventChanged(ev *calendar.Event, changeType string) string {
	var sb strings.Builder

	switch changeType {
	case "moved":
		sb.WriteString(fmt.Sprintf("🔄 *Встреча перенесена:* %s\n", EscMD(ev.Title)))
		sb.WriteString(fmt.Sprintf("📅 Новое время: *%s в %s*\n",
			ev.Start.Format("02\\.01"),
			ev.Start.Format("15:04"),
		))
	case "cancelled":
		sb.WriteString(fmt.Sprintf("❌ *Встреча отменена:* %s\n", EscMD(ev.Title)))
	case "new":
		sb.WriteString(fmt.Sprintf("🆕 *Новая встреча:* %s\n", EscMD(ev.Title)))
		sb.WriteString(fmt.Sprintf("📅 %s в %s\n",
			ev.Start.Format("02\\.01"),
			ev.Start.Format("15:04"),
		))
	default:
		sb.WriteString(fmt.Sprintf("✏️ *Изменена встреча:* %s\n", EscMD(ev.Title)))
		sb.WriteString(fmt.Sprintf("📅 %s в %s\n",
			ev.Start.Format("02\\.01"),
			ev.Start.Format("15:04"),
		))
	}

	sb.WriteString(formatLinks(ev))
	return sb.String()
}

func formatShort(num int, ev *calendar.Event) string {
	var sb strings.Builder

	timeStr := ev.Start.Format("15:04")
	if !ev.End.IsZero() {
		timeStr += "–" + ev.End.Format("15:04")
	}

	sb.WriteString(fmt.Sprintf("%d\\. *%s* \\(%s\\)\n", num, EscMD(ev.Title), EscMD(timeStr)))
	sb.WriteString(formatLinks(ev))
	sb.WriteString("\n")
	return sb.String()
}

func formatLinks(ev *calendar.Event) string {
	var sb strings.Builder

	if ev.MeetLink != "" {
		sb.WriteString(fmt.Sprintf("   📹 [Подключиться к Meet](%s)\n", ev.MeetLink))
	}
	for _, link := range ev.Links {
		if link != ev.MeetLink {
			sb.WriteString(fmt.Sprintf("   🔗 %s\n", link))
		}
	}
	return sb.String()
}

func russianDate(t time.Time) string {
	months := []string{
		"", "января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря",
	}
	return fmt.Sprintf("%d %s %d", t.Day(), months[t.Month()], t.Year())
}

// escMD экранирует спецсимволы для Telegram MarkdownV2
func EscMD(s string) string {
	r := strings.NewReplacer(
		"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]",
		"(", "\\(", ")", "\\)", "~", "\\~", "`", "\\`",
		">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
		"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}",
		".", "\\.", "!", "\\!",
	)
	return r.Replace(s)
}

func stripHTML(s string) string {
	var out strings.Builder
	inTag := false
	for _, c := range s {
		if c == '<' {
			inTag = true
		} else if c == '>' {
			inTag = false
			out.WriteRune(' ')
		} else if !inTag {
			out.WriteRune(c)
		}
	}
	return strings.Join(strings.Fields(out.String()), " ")
}

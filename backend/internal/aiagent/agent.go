package aiagent

// Agent orchestrates two layers of conversational behaviour in MAX:
//
// Layer 1 — rule-based bot: deterministic state machine for greeting,
// FAQ keywords, service picking, lead capture and handoff to the operator.
//
// Layer 2 — AI layer: optional LLM-backed reply grounded in the editable
// knowledge base, invoked only when the rule engine can't confidently answer.
// It is never given full autonomy: symptom/diagnosis answers are disclaimed,
// it cannot quote a final price, and complex cases escalate to a human.
//
// Both layers are exposed through one Agent.Handle call used by the MAX
// webhook handler. The agent returns the text to send back and an optional
// structured lead payload to persist.

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/timemachine-auto/timemachine/internal/config"
	kbrepo "github.com/timemachine-auto/timemachine/internal/domain/knowledge"
)

// Answer is the result of processing one inbound message.
type Answer struct {
	Text            string     // message to send back to the user
	Buttons         []string   // optional quick-reply labels (rule layer only)
	ButtonLabelHint []string   // hint labels the MAX layer may render as buttons
	Lead            *LeadDraft // if non-nil, a qualified lead was captured
	Escalate        bool       // request human operator takeover
	Done            bool       // conversation completed (lead saved)
	Disclaimer      bool       // attach the safety disclaimer
}

// LeadDraft is an in-progress lead assembled from the chat.
type LeadDraft struct {
	Name    string
	Phone   string
	Service string
	Symptom string
}

// Seat is the per-chat state stored between turns.
type Seat struct {
	Step    string    `json:"step"`
	Name    string    `json:"name,omitempty"`
	Phone   string    `json:"phone,omitempty"`
	Service string    `json:"service,omitempty"`
	Symptom string    `json:"symptom,omitempty"`
	Updated time.Time `json:"updated"`
}

const (
	StepStart   = "start"
	StepName    = "name"
	StepPhone   = "phone"
	StepService = "service"
	StepSymptom = "symptom"
	StepConfirm = "confirm"
)

var (
	symptomRE = regexp.MustCompile(`(?i)(стук|греет|не завод|тормоз|вибрац|шум|горит лампа|трясёт)`)
	urgentRE  = regexp.MustCompile(`(?i)(не тормозит|стукан|стакан бо|дым|течь|течь масл|перегрев|перегрел|горит ламп|горит чек|горит ошибка)`)
	handoffRE = regexp.MustCompile(`(?i)(оператор|человек|менеджер|сотрудник)`)
	priceRE   = regexp.MustCompile(`(?i)(цена|стоит|сколько)`)
)

// Agent depends only on small interfaces so it is testable without a DB.
type Agent struct {
	cfg   config.AIConfig
	log   *slog.Logger
	kb    KB
	rules *RuleEngine
	ai    AIBackend
}

type KB interface {
	Search(ctx context.Context, query string, limit int) ([]kbrepo.Item, error)
}

type AIBackend interface {
	Ask(ctx context.Context, system, user string) (string, error)
}

// NewAgent builds an agent. ai may be nil when AI is disabled.
func NewAgent(cfg config.AIConfig, log *slog.Logger, kb KB, ai AIBackend) *Agent {
	return &Agent{cfg: cfg, log: log, kb: kb, rules: NewRuleEngine(), ai: ai}
}

// Handle runs the inbound message through rule layer first; on miss and when
// AI is enabled, consults the LLM grounded in the KB.
func (a *Agent) Handle(ctx context.Context, seat *Seat, text string) Answer {
	text = strings.TrimSpace(text)
	seat.Updated = time.Now()

	if handoffRE.MatchString(text) {
		return Answer{Text: "Переключаю вас на оператора 🔧 Подождите немного.", Escalate: true}
	}

	// Rule-based conversation flow.
	if a.rules != nil {
		if ans := a.rules.Advance(seat, text); ans.Matched {
			return a.decorate(ans)
		}
	}

	// Urgent safety path before any AI.
	if urgentRE.MatchString(text) {
		return Answer{
			Text:       "Это может быть вопрос безопасности. Лучше опишите вкратце симптомы оператору — передаю вас специалисту.",
			Escalate:   true,
			Disclaimer: true,
		}
	}

	// Price question → direct to prices page, do not invent numbers.
	if priceRE.MatchString(text) {
		return Answer{
			Text:       "Актуальные цены публикуем в разделе «Прайс-лист» на сайте. Точную смету сформируем после осмотра — ставить цену по переписке некорректно.",
			Disclaimer: true,
		}
	}

	// Try KB-grounded AI answer.
	if a.cfg.Enabled && a.ai != nil {
		if ans, ok := a.tryAI(ctx, text); ok {
			return ans
		}
	}

	// Fallback: try KB directly as a knowledge snippet.
	if items, _ := a.kb.Search(ctx, text, 1); len(items) > 0 {
		return Answer{Text: items[0].Body, Disclaimer: true}
	}

	return Answer{
		Text:            "Подскажите, какое направление вас интересует? Я могу помочь: записать на ремонт, подобрать услугу, ответить по часто задаваемым вопросам. Напишите «оператор», чтобы переключиться на человека.",
		ButtonLabelHint: []string{"Записаться", "Прайс", "Контакты"},
		Disclaimer:      false,
	}
}

func (a *Agent) tryAI(ctx context.Context, user string) (Answer, bool) {
	items, err := a.kb.Search(ctx, user, 4)
	if err != nil || len(items) == 0 {
		return Answer{}, false
	}
	var context strings.Builder
	for _, it := range items {
		context.WriteString("- ")
		context.WriteString(it.Title)
		context.WriteString(": ")
		context.WriteString(it.Body)
		context.WriteString("\n")
	}
	system := fmt.Sprintf(`Ты — ассистент автомастерской «Машина времени». Отвечай кратко по-русски, по делу, на основе базы знаний ниже.
Ограничения:
- НЕ ставь диагноз автомобилю и не обещай точную цену до осмотра.
- Если случай сложный или опасный — порекомендуй приехать на осмотр или переключиться на оператора (слово «оператор»).
- Не выдумывай услуги и цены, которых нет в базе.

База знаний:
%s
Сегодняшняя дата: %s`, context.String(), time.Now().Format("02.01.2006"))

	out, err := a.ai.Ask(ctx, system, user)
	if err != nil {
		a.log.Error("ai.ask failed", "err", err)
		return Answer{}, false
	}
	if out == "" {
		return Answer{}, false
	}
	return Answer{Text: strings.TrimSpace(out), Disclaimer: true}, true
}

func (a *Agent) decorate(r RuleResult) Answer {
	ans := Answer{Text: r.Text}
	ans.ButtonLabelHint = r.Buttons
	if r.Lead != nil {
		ans.Lead = r.Lead
	}
	if r.Done {
		ans.Done = true
		ans.Disclaimer = true
	}
	return ans
}

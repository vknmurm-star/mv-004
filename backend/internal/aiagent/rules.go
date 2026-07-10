package aiagent

import (
	"regexp"
	"strings"
)

// RuleEngine implements a deterministic conversation for lead capture.
// It is intentionally simple and predictable so the bot always behaves the
// same way; the AI layer only augments, never replaces the safety-critical
// flow (urgent symptoms, escalation, price disclaimers).
type RuleEngine struct{}

func NewRuleEngine() *RuleEngine { return &RuleEngine{} }

// RuleResult is what the rule layer produces for a single turn.
type RuleResult struct {
	Matched bool
	Text    string
	Buttons []string
	Lead    *LeadDraft
	Done    bool
}

var phoneRE = regexp.MustCompile(`[^\d+]`)

// Advance runs one step of the rule state machine.
func (e *RuleEngine) Advance(s *Seat, text string) RuleResult {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "привет" || t == "здравствуйте" || t == "/start" || t == "старт" {
		s.Step = StepStart
		return welcome(s)
	}
	if t == "записаться" || t == "запись" {
		s.Step = StepName
		return RuleResult{
			Matched: true,
			Text:    "Отлично, запишем вас. Как вас зовут?",
			Buttons: []string{"Сначала хочу задать вопрос"},
		}
	}

	switch s.Step {
	case StepStart:
		if t == "запись" || strings.Contains(t, "записат") {
			s.Step = StepName
			return RuleResult{Matched: true, Text: "Как вас зовут?"}
		}
		return RuleResult{Matched: false}
	case StepName:
		if len(text) < 2 {
			return RuleResult{Matched: true, Text: "Имя чуть длиннее, пожалуйста :)"}
		}
		s.Name = text
		s.Step = StepPhone
		return RuleResult{Matched: true, Text: "Спасибо, " + text + ". Оставьте телефон для связи:"}
	case StepPhone:
		phone := phoneRE.ReplaceAllString(text, "")
		if len(phone) < 10 {
			return RuleResult{Matched: true, Text: "Похоже, телефон введён не полностью. Попробуйте ещё раз."}
		}
		s.Phone = phone
		s.Step = StepService
		return RuleResult{Matched: true, Text: "Какая услуга нужна? (диагностика, ТО, тормоза, подвеска, электрика, кузов и т.д.)"}
	case StepService:
		if strings.TrimSpace(text) == "" {
			return RuleResult{Matched: true, Text: "Напишите услугу или категорию, пожалуйста."}
		}
		s.Service = text
		s.Step = StepSymptom
		return RuleResult{Matched: true, Text: "Опишите кратко, что беспокоит в авто (необязательно). Можете написать «далее»."}
	case StepSymptom:
		if !strings.EqualFold(text, "далее") {
			s.Symptom = text
		}
		s.Step = StepConfirm
		return RuleResult{
			Matched: true,
			Text:    "Готово! Передаю заявку менеджеру — он подтвердит дату и время в рабочие часы. Спасибо!",
			Lead: &LeadDraft{
				Name:    s.Name,
				Phone:   s.Phone,
				Service: s.Service,
				Symptom: s.Symptom,
			},
			Done: true,
		}
	}
	return RuleResult{Matched: false}
}

func welcome(s *Seat) RuleResult {
	return RuleResult{
		Matched: true,
		Text: `Здравствуйте! Это «Машина времени» 🕒
Я помогу записаться на ремонт, подобрать услугу или ответить на частые вопросы.
Внимание: точный диагноз ставится только после осмотра авто мастером.`,
		Buttons: []string{"Записаться", "Прайс", "Контакты", "Оператор"},
	}
}

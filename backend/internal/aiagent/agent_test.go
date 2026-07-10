package aiagent

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/timemachine-auto/timemachine/internal/config"
	kbrepo "github.com/timemachine-auto/timemachine/internal/domain/knowledge"
)

func testAgentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func aiCfgDisabled() config.AIConfig { return config.AIConfig{Enabled: false} }

func TestRuleEngine_LeadCaptureFlow(t *testing.T) {
	e := NewRuleEngine()
	seat := Seat{Step: StepStart}

	if r := e.Advance(&seat, "записаться"); !r.Matched {
		t.Fatalf("entry to flow not matched: %+v", r)
	}
	if seat.Step != StepName {
		t.Fatalf("step = %q, want name", seat.Step)
	}
	if r := e.Advance(&seat, "Анна"); !r.Matched || seat.Name != "Анна" {
		t.Fatalf("name capture failed: %+v seat=%+v", r, seat)
	}
	if seat.Step != StepPhone {
		t.Fatalf("step = %q, want phone", seat.Step)
	}
	// bad phone must not be accepted
	if r := e.Advance(&seat, "abc"); !r.Matched || seat.Phone != "" {
		t.Errorf("bad phone should not be accepted, seat=%+v", seat)
	}
	// good phone
	if r := e.Advance(&seat, "+7 916 1234567"); !r.Matched || seat.Phone == "" {
		t.Errorf("good phone rejected: %+v seat=%+v", r, seat)
	}
	if seat.Step != StepService {
		t.Fatalf("step = %q, want service", seat.Step)
	}
	// service
	if r := e.Advance(&seat, "передние тормоза"); !r.Matched {
		t.Errorf("service step rejected: %+v", r)
	}
	if seat.Service != "передние тормоза" {
		t.Errorf("service not recorded: %q", seat.Service)
	}
	if seat.Step != StepSymptom {
		t.Fatalf("step = %q, want symptom", seat.Step)
	}
	// symptom → done, lead produced
	r := e.Advance(&seat, "скрип при торможении")
	if !r.Matched || !r.Done || r.Lead == nil {
		t.Fatalf("expected Done+Lead at confirm, got %+v", r)
	}
	if r.Lead.Name != "Анна" || r.Lead.Phone == "" || r.Lead.Service != "передние тормоза" {
		t.Errorf("lead content wrong: %+v", r.Lead)
	}
	if r.Lead.Symptom != "скрип при торможении" {
		t.Errorf("symptom not captured: %q", r.Lead.Symptom)
	}
}

func TestRuleEngine_SymptomSkip(t *testing.T) {
	e := NewRuleEngine()
	seat := Seat{Step: StepSymptom}
	r := e.Advance(&seat, "далее")
	if !r.Matched || !r.Done || r.Lead == nil {
		t.Fatalf("expected lead when skipping symptom, got %+v", r)
	}
	if r.Lead.Symptom != "" {
		t.Errorf("symptom should be empty on skip, got %q", r.Lead.Symptom)
	}
}

type noKB struct{}

func (noKB) Search(ctx context.Context, q string, limit int) ([]kbrepo.Item, error) {
	return nil, nil
}

func TestAgent_UrgentEscalates(t *testing.T) {
	a := NewAgent(aiCfgDisabled(), testAgentLogger(), noKB{}, nil)
	ans := a.Handle(context.Background(), &Seat{Step: StepStart}, "машина не тормозит совсем, педаль мягкая")
	if !ans.Escalate {
		t.Errorf("urgent symptoms must escalate, got %+v", ans)
	}
	if !ans.Disclaimer {
		t.Error("urgent answer should carry disclaimer")
	}
}

func TestAgent_HandoffKeyword(t *testing.T) {
	a := NewAgent(aiCfgDisabled(), testAgentLogger(), noKB{}, nil)
	ans := a.Handle(context.Background(), &Seat{Step: StepName}, "оператор")
	if !ans.Escalate {
		t.Errorf("handoff keyword must escalate, got %+v", ans)
	}
}

func TestAgent_PriceQuestionNeverInventsNumbers(t *testing.T) {
	a := NewAgent(aiCfgDisabled(), testAgentLogger(), noKB{}, nil)
	ans := a.Handle(context.Background(), &Seat{Step: StepStart}, "сколько стоит замена масла?")
	if !ans.Disclaimer {
		t.Errorf("price answer must disclaim, got %+v", ans)
	}
	// Must NOT contain arabic digits (i.e. never quotes a specific price).
	if containsDigit(ans.Text) {
		t.Errorf("price answer must not include digits, got: %q", ans.Text)
	}
}

func containsDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func TestAgent_KBSnippetFallbackRepeatsBody(t *testing.T) {
	mem := &memKB{items: []kbrepo.Item{{Title: "Адрес", Body: "Москва, ул. Гаражная 12"}}}
	a := NewAgent(aiCfgDisabled(), testAgentLogger(), mem, nil)
	ans := a.Handle(context.Background(), &Seat{Step: StepStart}, "адрес")
	if ans.Text != "Москва, ул. Гаражная 12" {
		t.Errorf("expected KB body fallback, got %q", ans.Text)
	}
	if !ans.Disclaimer {
		t.Error("KB answer should carry disclaimer")
	}
}

type memKB struct {
	items []kbrepo.Item
}

func (m *memKB) Search(ctx context.Context, q string, limit int) ([]kbrepo.Item, error) {
	if limit < 1 || limit > len(m.items) {
		limit = len(m.items)
	}
	return m.items[:limit], nil
}

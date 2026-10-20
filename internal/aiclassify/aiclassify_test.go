package aiclassify

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	camodels "github.com/abhinavxd/libredesk/internal/custom_attribute/models"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type fakeAI struct {
	configured bool
	reply      string
	err        error
	calls      int
	system     string
	user       string
}

func (f *fakeAI) CompletionConfigured() bool { return f.configured }

func (f *fakeAI) CompletionRaw(_ context.Context, system, user string) (string, error) {
	f.calls++
	f.system, f.user = system, user
	return f.reply, f.err
}

type fakeAttrs struct{ attrs []camodels.CustomAttribute }

func (f fakeAttrs) GetAll(string) ([]camodels.CustomAttribute, error) { return f.attrs, nil }

type fakeConvs struct {
	msgs   []cmodels.Message
	setKey string
	setVal string
	sets   int
}

func (f *fakeConvs) GetAllConversationMessages(string, *bool, []string, int) ([]cmodels.Message, error) {
	return f.msgs, nil
}

func (f *fakeConvs) SetConversationCustomAttributeIfEmpty(_, key, value string) (bool, error) {
	f.sets++
	f.setKey, f.setVal = key, value
	return true, nil
}

var questionType = camodels.CustomAttribute{
	Key:      "question_type",
	Name:     "Question type",
	DataType: "list",
	Values:   []string{"bug", "user_error", "feature_request", "meeting_scheduling", "billing", "integration_sync", "other"},
}

func newTest(ai *fakeAI) (*Classifier, *fakeConvs) {
	lo := logf.New(logf.Opts{})
	convs := &fakeConvs{msgs: []cmodels.Message{{SenderType: cmodels.SenderTypeContact, TextContent: "The Jobber sync stopped importing my expenses."}}}
	return New(context.Background(), ai, fakeAttrs{attrs: []camodels.CustomAttribute{questionType}}, convs, &lo), convs
}

func conv() cmodels.Conversation {
	return cmodels.Conversation{UUID: "c1", Subject: null.StringFrom("Sync broken"), CustomAttributes: json.RawMessage(`{}`)}
}

func TestRunSetsAllowedValue(t *testing.T) {
	ai := &fakeAI{configured: true, reply: "```json\n{\"value\": \"Integration_Sync\"}\n```"}
	c, convs := newTest(ai)

	if err := c.Run(context.Background(), conv(), "question_type", "Sync problems are integration_sync."); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if convs.sets != 1 || convs.setKey != "question_type" || convs.setVal != "integration_sync" {
		t.Fatalf("want question_type=integration_sync set once, got %d sets %q=%q", convs.sets, convs.setKey, convs.setVal)
	}
	if !strings.Contains(ai.system, "- feature_request") || !strings.Contains(ai.system, "Sync problems are integration_sync.") {
		t.Fatalf("system prompt missing values or instructions: %s", ai.system)
	}
	if !strings.Contains(ai.user, "Subject: Sync broken") || !strings.Contains(ai.user, "Jobber sync stopped") {
		t.Fatalf("user prompt missing subject or message: %s", ai.user)
	}
}

func TestRunIgnoresValueOutsideList(t *testing.T) {
	for _, reply := range []string{`{"value": "outage"}`, `outage`, `{"label": "bug"}`, `{"value": "none"}`} {
		ai := &fakeAI{configured: true, reply: reply}
		c, convs := newTest(ai)
		if err := c.Run(context.Background(), conv(), "question_type", ""); err != nil {
			t.Fatalf("reply %q: Run: %v", reply, err)
		}
		if convs.sets != 0 {
			t.Fatalf("reply %q: attribute must not be set, got %q", reply, convs.setVal)
		}
	}
}

func TestRunProviderError(t *testing.T) {
	ai := &fakeAI{configured: true, err: errors.New("rate limited")}
	c, convs := newTest(ai)
	if err := c.Run(context.Background(), conv(), "question_type", ""); err == nil {
		t.Fatal("want provider error returned")
	}
	if convs.sets != 0 {
		t.Fatal("attribute must not be set on provider error")
	}
}

func TestRunProviderNotConfigured(t *testing.T) {
	ai := &fakeAI{configured: false, reply: `{"value": "bug"}`}
	c, convs := newTest(ai)
	if err := c.Run(context.Background(), conv(), "question_type", ""); err != nil {
		t.Fatalf("not configured must be a no-op, got %v", err)
	}
	if ai.calls != 0 || convs.sets != 0 {
		t.Fatalf("want no provider call and no write, got %d calls %d sets", ai.calls, convs.sets)
	}
}

func TestRunSkipsAttributeAlreadySet(t *testing.T) {
	ai := &fakeAI{configured: true, reply: `{"value": "bug"}`}
	c, convs := newTest(ai)
	cv := conv()
	cv.CustomAttributes = json.RawMessage(`{"question_type": "billing"}`)
	if err := c.Run(context.Background(), cv, "question_type", ""); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if ai.calls != 0 || convs.sets != 0 {
		t.Fatal("an attribute an agent already set must be left alone")
	}
}

func TestRunUnknownAttribute(t *testing.T) {
	ai := &fakeAI{configured: true, reply: `{"value": "bug"}`}
	c, _ := newTest(ai)
	if err := c.Run(context.Background(), conv(), "missing", ""); err == nil {
		t.Fatal("want error for an unknown attribute key")
	}
}

func TestParseChoiceText(t *testing.T) {
	attr := camodels.CustomAttribute{DataType: "text"}
	if v, ok := parseChoice(`{"value": "  Login\n page  "}`, attr); !ok || v != "Login page" {
		t.Fatalf("got %q %v", v, ok)
	}
	long := `{"value": "` + strings.Repeat("a", 500) + `"}`
	if v, ok := parseChoice(long, attr); !ok || len(v) != maxTextValueRunes {
		t.Fatalf("want text capped at %d, got %d", maxTextValueRunes, len(v))
	}
}

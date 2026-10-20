package aiclassify

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
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
	attrs    string
	msgs     []cmodels.Message
	msgLimit int
	setKey   string
	setVal   any
	sets     int
}

func (f *fakeConvs) GetConversation(int, string, string) (cmodels.Conversation, error) {
	return cmodels.Conversation{UUID: "c1", Subject: null.StringFrom("Sync broken"), CustomAttributes: json.RawMessage(f.attrs)}, nil
}

func (f *fakeConvs) GetFirstIncomingMessages(_ string, limit int) ([]cmodels.Message, error) {
	f.msgLimit = limit
	return f.msgs, nil
}

func (f *fakeConvs) SetConversationCustomAttributeIfEmpty(_, key string, value any) (bool, error) {
	f.sets++
	f.setKey, f.setVal = key, value
	return true, nil
}

// The question_type values and descriptions Clyr used in Pylon.
var pylonValues = []string{"Meeting Scheduling", "Bug", "Feature Request", "User Error", "Customer Love"}

var pylonDescriptions = map[string]string{
	"Bug":        "The conversation is about the customer reporting a bug or something not working with the product.",
	"User Error": "The conversation is about the vendor telling the customer that they are using the product incorrectly.",
}

func attr(dataType string) camodels.CustomAttribute {
	return camodels.CustomAttribute{Key: "question_type", Name: "Question type", DataType: dataType, Values: pylonValues}
}

func newTest(ai *fakeAI, a camodels.CustomAttribute) (*Classifier, *fakeConvs) {
	lo := logf.New(logf.Opts{})
	convs := &fakeConvs{attrs: `{}`, msgs: []cmodels.Message{{SenderType: cmodels.SenderTypeContact, TextContent: "The Jobber sync stopped importing my expenses."}}}
	return New(context.Background(), ai, fakeAttrs{attrs: []camodels.CustomAttribute{a}}, convs, &lo), convs
}

func params(t *testing.T, value ...string) Params {
	t.Helper()
	p, err := ParseParams(value)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRunSetsAllowedValue(t *testing.T) {
	ai := &fakeAI{configured: true, reply: "```json\n{\"value\": \"bug\"}\n```"}
	c, convs := newTest(ai, attr(DataTypeList))
	desc, _ := json.Marshal(pylonDescriptions)

	if err := c.Run(context.Background(), "c1", params(t, "question_type", "Prefer Bug when unsure.", string(desc))); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if convs.sets != 1 || convs.setKey != "question_type" || convs.setVal != "Bug" {
		t.Fatalf("want question_type=Bug set once, got %d sets %q=%v", convs.sets, convs.setKey, convs.setVal)
	}
	if convs.msgLimit != maxClassifyMsgs {
		t.Fatalf("want the first %d messages requested, got %d", maxClassifyMsgs, convs.msgLimit)
	}
	for _, want := range []string{"- Bug: The conversation is about the customer reporting a bug", "- Customer Love\n", "Prefer Bug when unsure.", `{"value"`} {
		if !strings.Contains(ai.system, want) {
			t.Fatalf("system prompt missing %q:\n%s", want, ai.system)
		}
	}
	if !strings.Contains(ai.user, "Subject: Sync broken") || !strings.Contains(ai.user, "Jobber sync stopped") {
		t.Fatalf("user prompt missing subject or message: %s", ai.user)
	}
}

func TestRunMultiSelect(t *testing.T) {
	cases := []struct {
		reply string
		want  any // nil = nothing set
	}{
		{`{"values": ["bug", "User Error", "Bug"]}`, []string{"Bug", "User Error"}},
		{`{"values": ["Bug", "outage"]}`, []string{"Bug"}},
		{`{"values": []}`, nil},
		{`{"values": ["none"]}`, nil},
		{`{"values": ["outage"]}`, nil},
		{`{"value": "Bug"}`, nil},
	}
	for _, tc := range cases {
		ai := &fakeAI{configured: true, reply: tc.reply}
		c, convs := newTest(ai, attr(DataTypeMultiSelect))
		if err := c.Run(context.Background(), "c1", params(t, "question_type")); err != nil {
			t.Fatalf("reply %s: Run: %v", tc.reply, err)
		}
		if tc.want == nil {
			if convs.sets != 0 {
				t.Fatalf("reply %s: want nothing set, got %v", tc.reply, convs.setVal)
			}
			continue
		}
		if !reflect.DeepEqual(convs.setVal, tc.want) {
			t.Fatalf("reply %s: got %v, want %v", tc.reply, convs.setVal, tc.want)
		}
	}
	ai := &fakeAI{configured: true, reply: `{"values": []}`}
	c, _ := newTest(ai, attr(DataTypeMultiSelect))
	_ = c.Run(context.Background(), "c1", params(t, "question_type"))
	if !strings.Contains(ai.system, `{"values": [`) {
		t.Fatalf("multi-select prompt must ask for a values array:\n%s", ai.system)
	}
}

func TestRunIgnoresValueOutsideList(t *testing.T) {
	for _, reply := range []string{`{"value": "outage"}`, `outage`, `{"label": "Bug"}`, `{"value": "none"}`} {
		ai := &fakeAI{configured: true, reply: reply}
		c, convs := newTest(ai, attr(DataTypeList))
		if err := c.Run(context.Background(), "c1", params(t, "question_type")); err != nil {
			t.Fatalf("reply %q: Run: %v", reply, err)
		}
		if convs.sets != 0 {
			t.Fatalf("reply %q: attribute must not be set, got %v", reply, convs.setVal)
		}
	}
}

func TestRunProviderError(t *testing.T) {
	ai := &fakeAI{configured: true, err: errors.New("rate limited")}
	c, convs := newTest(ai, attr(DataTypeList))
	if err := c.Run(context.Background(), "c1", params(t, "question_type")); err == nil {
		t.Fatal("want provider error returned")
	}
	if convs.sets != 0 {
		t.Fatal("attribute must not be set on provider error")
	}
}

func TestRunProviderNotConfigured(t *testing.T) {
	ai := &fakeAI{configured: false, reply: `{"value": "Bug"}`}
	c, convs := newTest(ai, attr(DataTypeList))
	if err := c.Run(context.Background(), "c1", params(t, "question_type")); err != nil {
		t.Fatalf("not configured must be a no-op, got %v", err)
	}
	if ai.calls != 0 || convs.sets != 0 {
		t.Fatalf("want no provider call and no write, got %d calls %d sets", ai.calls, convs.sets)
	}
}

func TestRunSkipsAttributeAlreadySet(t *testing.T) {
	for _, stored := range []string{`{"question_type": "Bug"}`, `{"question_type": ["Bug"]}`} {
		ai := &fakeAI{configured: true, reply: `{"value": "Bug"}`}
		c, convs := newTest(ai, attr(DataTypeList))
		convs.attrs = stored
		if err := c.Run(context.Background(), "c1", params(t, "question_type")); err != nil {
			t.Fatalf("Run: %v", err)
		}
		if ai.calls != 0 || convs.sets != 0 {
			t.Fatalf("stored %s: an attribute an agent already set must be left alone", stored)
		}
	}
}

func TestRunUnknownAttribute(t *testing.T) {
	ai := &fakeAI{configured: true, reply: `{"value": "Bug"}`}
	c, _ := newTest(ai, attr(DataTypeList))
	if err := c.Run(context.Background(), "c1", params(t, "missing")); err == nil {
		t.Fatal("want error for an unknown attribute key")
	}
}

func TestParseParams(t *testing.T) {
	if _, err := ParseParams(nil); err == nil {
		t.Fatal("want error without a key")
	}
	if _, err := ParseParams([]string{"question_type", "", "not json"}); err == nil {
		t.Fatal("want error for malformed descriptions")
	}
	p, err := ParseParams([]string{" question_type ", " hint ", `{"Bug": "broken"}`})
	if err != nil || p.Key != "question_type" || p.Instructions != "hint" || p.Descriptions["Bug"] != "broken" {
		t.Fatalf("got %+v %v", p, err)
	}
}

func TestHasValue(t *testing.T) {
	for raw, want := range map[string]bool{
		`{}`: false, `{"k": null}`: false, `{"k": "  "}`: false, `{"k": []}`: false,
		`{"k": "Bug"}`: true, `{"k": ["Bug"]}`: true, `{"k": 0}`: true,
	} {
		if got := HasValue(json.RawMessage(raw), "k"); got != want {
			t.Fatalf("HasValue(%s) = %v, want %v", raw, got, want)
		}
	}
}

func TestParseChoiceText(t *testing.T) {
	a := camodels.CustomAttribute{DataType: DataTypeText}
	if v, ok := parseChoice(`{"value": "  Login\n page  "}`, a); !ok || v != "Login page" {
		t.Fatalf("got %v %v", v, ok)
	}
	long := `{"value": "` + strings.Repeat("a", 500) + `"}`
	if v, ok := parseChoice(long, a); !ok || len(v.(string)) != maxTextValueRunes {
		t.Fatalf("want text capped at %d, got %v", maxTextValueRunes, v)
	}
}

func TestClassifyCoalescesRepeatedEvents(t *testing.T) {
	ai := &fakeAI{configured: true, reply: `{"value": "Bug"}`}
	c, _ := newTest(ai, attr(DataTypeList))
	// Hold every slot so queued work stays in flight.
	for i := 0; i < maxConcurrent; i++ {
		c.sem <- struct{}{}
	}
	c.Classify("c1", []string{"question_type"})
	c.Classify("c1", []string{"question_type"})
	c.Classify("c2", []string{"question_type"})
	c.inflightMu.Lock()
	n := len(c.inflight)
	c.inflightMu.Unlock()
	if n != 2 {
		t.Fatalf("want one in-flight entry per conversation and key, got %d", n)
	}
}

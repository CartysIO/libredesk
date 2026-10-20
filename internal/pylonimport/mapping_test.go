package pylonimport

import (
	"reflect"
	"testing"
	"time"
)

func TestMapStatus(t *testing.T) {
	cases := map[string]string{
		"closed":              StatusResolved,
		"waiting_on_customer": StatusOnCustomer,
		"on_hold":             StatusSnoozed,
		"new":                 StatusOpen,
		"waiting_on_you":      StatusOnYou,
		"something_else":      StatusOpen,
	}
	for in, want := range cases {
		if got := MapStatus(in); got != want {
			t.Errorf("MapStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestKeepTag(t *testing.T) {
	for tag, want := range map[string]bool{
		"PR-1409":       true,
		"bug":           true,
		"loop-triaged":  false,
		"Loop-needs-hu": false,
		"  ":            false,
		"answer-ready":  true,
	} {
		if got := KeepTag(tag); got != want {
			t.Errorf("KeepTag(%q) = %v, want %v", tag, got, want)
		}
	}
}

func TestQuestionType(t *testing.T) {
	fields := map[string]CustomField{"question_type": {Value: "", Values: []string{"bug", "user_error", "bug"}}}
	if got := QuestionType(fields); got != "bug, user_error" {
		t.Errorf("multi-select: got %q", got)
	}
	if got := QuestionType(map[string]CustomField{"question_type": {Value: "feature_request"}}); got != "feature_request" {
		t.Errorf("single value: got %q", got)
	}
	if got := QuestionType(nil); got != "" {
		t.Errorf("missing: got %q", got)
	}
}

func TestOrgExternalID(t *testing.T) {
	withLabel := Account{ID: "acc-1", ExternalIDs: []ExternalID{{Label: "other", ExternalID: "x"}, {Label: "clyr_team_id", ExternalID: " team_1 "}}}
	if got := OrgExternalID(withLabel); got != "team_1" {
		t.Errorf("clyr_team_id: got %q", got)
	}
	withField := Account{ID: "acc-2", CustomFields: map[string]CustomField{"teamid": {Value: "team_2"}}}
	if got := OrgExternalID(withField); got != "team_2" {
		t.Errorf("teamid field: got %q", got)
	}
	if got := OrgExternalID(Account{ID: "acc-3"}); got != "acc-3" {
		t.Errorf("fallback: got %q", got)
	}
}

func TestSplitNameAndTitled(t *testing.T) {
	if f, l := SplitName("  Vayco   Stays Inc ", ""); f != "Vayco" || l != "Stays Inc" {
		t.Errorf("SplitName = %q %q", f, l)
	}
	if f, l := SplitName("", "josh@xl-rpm.com"); f != "josh" || l != "" {
		t.Errorf("SplitName from email = %q %q", f, l)
	}
	if f, _ := SplitName("", ""); f != "Unknown" {
		t.Errorf("SplitName empty = %q", f)
	}
	if got := Titled("jaleel"); got != "Jaleel" {
		t.Errorf("Titled = %q", got)
	}
}

func TestFileNameAndKey(t *testing.T) {
	u := "https://assets.usepylon.com/9cbc732f-de6d-4722-a4bc-56a5126abb86%2Fed4e5656-fc77-4b5d-a99f-d9723328d20b-unnamed.png?Expires=1&Signature=a"
	if got := FileName(u); got != "unnamed.png" {
		t.Errorf("FileName = %q", got)
	}
	if FileKey(u) != FileKey(u[:len(u)-1]+"b") {
		t.Error("FileKey must ignore the signature")
	}
	if FileKey(u) == FileKey("https://assets.usepylon.com/other.png") {
		t.Error("FileKey must differ for different files")
	}
	if got := FileName("https://files.example.com/a/report%20Q3.pdf"); got != "report Q3.pdf" {
		t.Errorf("FileName plain = %q", got)
	}
}

func TestSameThread(t *testing.T) {
	at := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	if !SameThread("A@x.com", "Cancel account", at, "a@x.com", "RE: cancel  account", at.Add(3*time.Hour)) {
		t.Error("same contact, subject and day should match")
	}
	if SameThread("a@x.com", "Cancel account", at, "b@x.com", "Cancel account", at) {
		t.Error("different contact must not match")
	}
	if SameThread("a@x.com", "Cancel account", at, "a@x.com", "Cancel account", at.Add(25*time.Hour)) {
		t.Error("more than a day apart must not match")
	}
	if SameThread("a@x.com", "", at, "a@x.com", "", at) {
		t.Error("empty subjects must not match")
	}
}

func TestReferenceFloor(t *testing.T) {
	if got := ReferenceFloor(4000, 3858); got != 4000 {
		t.Errorf("got %d", got)
	}
	if got := ReferenceFloor(4000, 4100); got != 4101 {
		t.Errorf("got %d", got)
	}
}

func TestPlanRenumbers(t *testing.T) {
	existing := []ExistingConversation{
		{ID: 1, ReferenceNumber: "100"}, // collides with Pylon #100: moves to the sequence
		{ID: 2, ReferenceNumber: "105"}, // duplicate of Pylon #3855: takes 3855
		{ID: 3, ReferenceNumber: "109"}, // in the Pylon range, though Pylon has no #109: moves too
		{ID: 4, ReferenceNumber: "3855", Merged: true},
		{ID: 5, ReferenceNumber: "4001"},
		{ID: 6, ReferenceNumber: "ABC-7"},
	}
	got := PlanRenumbers(existing, 3858, map[int]int{2: 3855})
	want := []Renumber{{ConversationID: 1, From: "100"}, {ConversationID: 3, From: "109"}, {ConversationID: 2, From: "105", To: "3855"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("PlanRenumbers = %+v, want %+v", got, want)
	}
}

func TestInlineFileURLs(t *testing.T) {
	file := "https://assets.usepylon.com/a%2Fimg.png?Expires=1&Signature=x"
	other := "https://assets.usepylon.com/a%2Fdoc.pdf?Expires=1"
	body := `<p>see</p><img alt="x" src="https://assets.usepylon.com/a%2Fimg.png?Expires=1&amp;Signature=y"><img src='https://elsewhere/p.png'>`
	got := InlineFileURLs(body, []string{file, other})
	want := map[string]string{file: "https://assets.usepylon.com/a%2Fimg.png?Expires=1&amp;Signature=y"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("InlineFileURLs = %v, want %v", got, want)
	}
}

func TestOrderedMessages(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	is := Issue{ID: "i", CreatedAt: t0, BodyHTML: "<p>hi</p>", Requester: &Ref{ID: "c"}, AttachmentURLs: []string{"https://f/1?s=1", "https://f/2?s=1"}}

	only := orderedMessages(is, nil)
	if len(only) != 1 || only[0].Author.Contact.ID != "c" || len(only[0].FileURLs) != 2 {
		t.Fatalf("issue body as message: %+v", only)
	}

	msgs := []Message{
		{ID: "b", Timestamp: t0.Add(2 * time.Minute)},
		{ID: "a", Timestamp: t0.Add(time.Minute), FileURLs: []string{"https://f/1?s=2"}},
	}
	got := orderedMessages(is, msgs)
	if got[0].ID != "a" || got[1].ID != "b" {
		t.Fatalf("order: %s %s", got[0].ID, got[1].ID)
	}
	if !reflect.DeepEqual(got[0].FileURLs, []string{"https://f/1?s=2", "https://f/2?s=1"}) {
		t.Errorf("issue files not on a message go on the first one: %v", got[0].FileURLs)
	}
	if len(msgs[1].FileURLs) != 1 {
		t.Error("orderedMessages must not change its input")
	}
}

func TestPlanAgent(t *testing.T) {
	im := &Importer{
		Opt:          Options{AgentAliases: map[string]string{"omer.menashe@gmail.com": "omer@clyr.io"}},
		agentByEmail: map[string]int{"omer@clyr.io": 2, "eran@clyr.io": 3},
		agentByPylon: map[string]int{},
	}
	if p := im.planAgent(User{ID: "u1", Name: "Omer Menashe", Email: "Omer.Menashe@gmail.com"}); p.existingID != 2 {
		t.Errorf("alias: %+v", p)
	}
	if p := im.planAgent(User{ID: "u2", Name: "eran", Email: "eran@clyr.io"}); p.existingID != 3 {
		t.Errorf("email: %+v", p)
	}
	p := im.planAgent(User{ID: "u3", Name: "jaleel", Email: "jaleel@clyr.io"})
	if p.existingID != 0 || p.first != "Jaleel" || p.last != "(former)" || p.email != "jaleel@clyr.io" {
		t.Errorf("former staff: %+v", p)
	}
	p = im.planAgent(User{ID: "u4", Name: "Backend Key"})
	if p.existingID != 0 || p.first != "Backend" || p.last != "Key (Pylon bot)" || p.email != "" {
		t.Errorf("API identity: %+v", p)
	}
}

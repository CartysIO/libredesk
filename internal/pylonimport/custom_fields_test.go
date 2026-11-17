package pylonimport

import "testing"

func TestMergeIssueCustomAttributesCollision(t *testing.T) {
	existing := map[string]string{"question_type": "native"}
	got := MergeIssueCustomAttributes(existing, map[string]CustomField{
		"question_type": {Values: []string{"bug"}},
		"chat_widget.os": {Value: "macos"},
	})
	if got["question_type"] != "native" {
		t.Fatalf("native key overwritten: %q", got["question_type"])
	}
	if got["pylon.question_type"] != "bug" {
		t.Fatalf("pylon prefix = %q", got["pylon.question_type"])
	}
	if got["chat_widget.os"] != "macos" {
		t.Fatalf("os = %q", got["chat_widget.os"])
	}
}

func TestInboxForSource(t *testing.T) {
	im := &Importer{emailInboxID: 2, chatInboxID: 3}
	if im.inboxForSource("chat_widget") != 3 || im.inboxForSource("email") != 2 || im.inboxForSource("form") != 2 {
		t.Fatal("inbox mapping")
	}
}
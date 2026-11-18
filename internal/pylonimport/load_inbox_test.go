package pylonimport

import (
	"bytes"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestLoadWithoutLivechatInboxFallsBackToEmail(t *testing.T) {
	db := testutil.NewDB(t, "pylon_load_inbox")
	var emailID int
	db.Get(&emailID, `INSERT INTO inboxes (name, channel, enabled) VALUES ('Only email', 'email', FALSE) RETURNING id`)
	im := &Importer{DB: db, Out: nil, Opt: Options{DryRun: true}}
	d := Data{Issues: []Issue{{ID: "i", Number: 1, Source: "chat_widget"}}}
	if err := im.load(d); err != nil {
		t.Fatal(err)
	}
	if im.emailInboxID != emailID || im.chatInboxID != emailID {
		t.Fatalf("inboxes email=%d chat=%d want %d", im.emailInboxID, im.chatInboxID, emailID)
	}
}

func TestLoadMultipleLivechatPicksEnabledFirst(t *testing.T) {
	db := testutil.NewDB(t, "pylon_load_inbox_multi")
	var emailID int
	db.Get(&emailID, `INSERT INTO inboxes (name, channel, enabled) VALUES ('E', 'email', FALSE) RETURNING id`)
	var olderID, enabledID int
	db.Get(&olderID, `INSERT INTO inboxes (name, channel, enabled) VALUES ('Old', 'livechat', FALSE) RETURNING id`)
	db.Get(&enabledID, `INSERT INTO inboxes (name, channel, enabled) VALUES ('Widget', 'livechat', TRUE) RETURNING id`)
	im := &Importer{DB: db, Opt: Options{DryRun: true}}
	if err := im.load(Data{}); err != nil {
		t.Fatal(err)
	}
	if im.emailInboxID != emailID {
		t.Fatalf("emailInboxID=%d want %d", im.emailInboxID, emailID)
	}
	if im.chatInboxID != enabledID {
		t.Fatalf("chatInboxID=%d want enabled inbox %d (older disabled=%d)", im.chatInboxID, enabledID, olderID)
	}
}

func TestLoadMultipleLivechatAllDisabledPicksLowestID(t *testing.T) {
	db := testutil.NewDB(t, "pylon_load_inbox_multi_disabled")
	db.Exec(`INSERT INTO inboxes (name, channel, enabled) VALUES ('E', 'email', FALSE)`)
	var lowID, highID int
	db.Get(&lowID, `INSERT INTO inboxes (name, channel, enabled) VALUES ('A', 'livechat', FALSE) RETURNING id`)
	db.Get(&highID, `INSERT INTO inboxes (name, channel, enabled) VALUES ('B', 'livechat', FALSE) RETURNING id`)
	im := &Importer{DB: db, Opt: Options{DryRun: true}}
	if err := im.load(Data{}); err != nil {
		t.Fatal(err)
	}
	if im.chatInboxID != lowID {
		t.Fatalf("chatInboxID=%d want lowest id %d (other=%d)", im.chatInboxID, lowID, highID)
	}
}

func TestLoadMultipleLivechatEmitsChoiceMessage(t *testing.T) {
	db := testutil.NewDB(t, "pylon_load_inbox_multi_msg")
	db.Exec(`INSERT INTO inboxes (name, channel, enabled) VALUES ('E', 'email', FALSE)`)
	var firstID, secondID int
	db.Get(&firstID, `INSERT INTO inboxes (name, channel, enabled) VALUES ('One', 'livechat', TRUE) RETURNING id`)
	db.Get(&secondID, `INSERT INTO inboxes (name, channel, enabled) VALUES ('Two', 'livechat', TRUE) RETURNING id`)
	var out bytes.Buffer
	im := &Importer{DB: db, Out: &out, Opt: Options{DryRun: true}}
	if err := im.load(Data{}); err != nil {
		t.Fatal(err)
	}
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}
	if im.chatInboxID != firstID {
		t.Fatalf("chatInboxID=%d want lower enabled id %d", im.chatInboxID, firstID)
	}
	got := out.String()
	if !strings.Contains(got, "found 2 livechat inboxes") || !strings.Contains(got, "using inbox") {
		t.Fatalf("stdout notice missing context: %q", got)
	}
}

func TestLoadMultipleLivechatRespectsExplicitID(t *testing.T) {
	db := testutil.NewDB(t, "pylon_load_inbox_multi_explicit")
	db.Exec(`INSERT INTO inboxes (name, channel, enabled) VALUES ('E', 'email', FALSE)`)
	var disabledID, enabledID int
	db.Get(&disabledID, `INSERT INTO inboxes (name, channel, enabled) VALUES ('Pick', 'livechat', FALSE) RETURNING id`)
	db.Get(&enabledID, `INSERT INTO inboxes (name, channel, enabled) VALUES ('Other', 'livechat', TRUE) RETURNING id`)
	im := &Importer{DB: db, Opt: Options{DryRun: true, ChatInboxID: disabledID}}
	if err := im.load(Data{}); err != nil {
		t.Fatal(err)
	}
	if im.chatInboxID != disabledID {
		t.Fatalf("chatInboxID=%d want explicit %d", im.chatInboxID, disabledID)
	}
	if im.chatInboxID == enabledID {
		t.Fatalf("explicit pick must not use enabled inbox %d", enabledID)
	}
}

package pylonimport

import (
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
	_ = enabledID
}

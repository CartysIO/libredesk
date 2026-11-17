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
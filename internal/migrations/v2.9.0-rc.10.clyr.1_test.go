package migrations

import (
	"testing"

	customAttribute "github.com/abhinavxd/libredesk/internal/custom_attribute"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestV2_9_0_RC10_CLYR1ChatContextAttributes(t *testing.T) {
	db := testutil.NewDB(t, "migration_clyr_chat_context")
	// An agent already made one of the definitions with a name of their own.
	db.MustExec(`DELETE FROM custom_attribute_definitions WHERE key LIKE 'chat\_%'`)
	db.MustExec(`INSERT INTO custom_attribute_definitions (name, description, applies_to, key, data_type, regex, regex_hint) VALUES ('Mine', 'mine', 'conversation', 'chat_app', 'text', '', '')`)

	// Running it twice proves it is idempotent, which matters because a fresh install runs every migration.
	for range 2 {
		if err := V2_9_0_RC10_CLYR1(db, nil, nil); err != nil {
			t.Fatalf("running migration: %v", err)
		}
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM custom_attribute_definitions WHERE applies_to = 'conversation' AND key LIKE 'chat\_%'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 8 {
		t.Fatalf("got %d chat definitions, want 8", count)
	}
	// The agent UI reads the definitions through the manager, which cannot scan a NULL regex.
	lo := logf.New(logf.Opts{Level: logf.FatalLevel})
	mgr, err := customAttribute.New(customAttribute.Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	listed, err := mgr.GetAll("conversation")
	if err != nil {
		t.Fatalf("listing conversation attributes: %v", err)
	}
	if len(listed) != 8 {
		t.Fatalf("manager lists %d conversation attributes, want 8", len(listed))
	}
	var name string
	if err := db.QueryRow(`SELECT name FROM custom_attribute_definitions WHERE key = 'chat_app'`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Mine" {
		t.Fatalf("existing definition was changed to %q", name)
	}
}

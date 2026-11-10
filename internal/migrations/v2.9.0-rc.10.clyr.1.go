package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_0_RC10_CLYR1 creates the conversation attribute definitions that show a live chat's visitor
// context in the agent sidebar: browser, operating system, device, page URL, User-Agent, timezone and app
// (Clyr fork). The chat widget fills them when a chat starts. A definition that exists is left alone.
func V2_9_0_RC10_CLYR1(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		INSERT INTO custom_attribute_definitions (name, description, applies_to, key, data_type, values, regex, regex_hint)
		VALUES
			('Browser', 'Browser name and major version of the visitor, recorded when a live chat starts', 'conversation', 'chat_browser', 'text', '{}', '', ''),
			('Operating System', 'Operating system of the visitor, recorded when a live chat starts', 'conversation', 'chat_operating_system', 'text', '{}', '', ''),
			('Device Type', 'desktop, mobile or tablet, recorded when a live chat starts', 'conversation', 'chat_device_type', 'text', '{}', '', ''),
			('Device Model', 'Device model of the visitor when the browser tells it, recorded when a live chat starts', 'conversation', 'chat_device_model', 'text', '{}', '', ''),
			('Page URL', 'URL of the page that has the chat widget, recorded when a live chat starts', 'conversation', 'chat_page_url', 'text', '{}', '', ''),
			('User Agent', 'Raw User-Agent header of the visitor, recorded when a live chat starts', 'conversation', 'chat_user_agent', 'text', '{}', '', ''),
			('User Timezone', 'IANA timezone of the visitor browser, recorded when a live chat starts', 'conversation', 'chat_user_timezone', 'text', '{}', '', ''),
			('App', 'App label that the host page passes to the chat widget, recorded when a live chat starts', 'conversation', 'chat_app', 'text', '{}', '', '')
		ON CONFLICT (key, applies_to) DO NOTHING;
	`)
	return err
}

package slackbridge

import (
	"database/sql"
	"embed"
	"errors"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/jmoiron/sqlx"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

const (
	directionIn  = "in"
	directionOut = "out"
)

type queries struct {
	GetThreadByConversation    *sqlx.Stmt `query:"get-thread-by-conversation"`
	GetThreadByTS              *sqlx.Stmt `query:"get-thread-by-ts"`
	InsertThread               *sqlx.Stmt `query:"insert-thread"`
	ClaimSlackMessage          *sqlx.Stmt `query:"claim-slack-message"`
	SetSlackMessageMessageID   *sqlx.Stmt `query:"set-slack-message-message-id"`
	InsertOutgoingSlackMessage *sqlx.Stmt `query:"insert-outgoing-slack-message"`
	MessageIsBridged           *sqlx.Stmt `query:"message-is-bridged"`
}

// thread links a conversation to its Slack thread.
type thread struct {
	ConversationID   int    `db:"conversation_id"`
	ChannelID        string `db:"channel_id"`
	ThreadTS         string `db:"thread_ts"`
	ConversationUUID string `db:"conversation_uuid"`
}

// store keeps the bridge's mapping between Libredesk and Slack in Postgres.
type store struct {
	q queries
}

func newStore(db *sqlx.DB) (*store, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		return nil, err
	}
	return &store{q: q}, nil
}

// threadByConversation returns the conversation's thread, or ok=false when it has none.
func (s *store) threadByConversation(conversationID int) (thread, bool, error) {
	var t thread
	err := s.q.GetThreadByConversation.Get(&t, conversationID)
	if errors.Is(err, sql.ErrNoRows) {
		return t, false, nil
	}
	return t, err == nil, err
}

// threadByTS returns the thread with this root ts in the channel, or ok=false.
func (s *store) threadByTS(channelID, threadTS string) (thread, bool, error) {
	var t thread
	err := s.q.GetThreadByTS.Get(&t, channelID, threadTS)
	if errors.Is(err, sql.ErrNoRows) {
		return t, false, nil
	}
	return t, err == nil, err
}

func (s *store) insertThread(conversationID int, channelID, threadTS string) error {
	_, err := s.q.InsertThread.Exec(conversationID, channelID, threadTS)
	return err
}

// claim records an incoming Slack message. ok=false means it was already handled.
func (s *store) claim(channelID, ts string) (id int, ok bool, err error) {
	err = s.q.ClaimSlackMessage.Get(&id, channelID, ts, directionIn)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return id, err == nil, err
}

func (s *store) setMessageID(claimID, messageID int) error {
	_, err := s.q.SetSlackMessageMessageID.Exec(claimID, messageID)
	return err
}

func (s *store) insertOutgoing(channelID, ts string, messageID int) error {
	_, err := s.q.InsertOutgoingSlackMessage.Exec(channelID, ts, messageID)
	return err
}

func (s *store) isBridged(messageID int) (bool, error) {
	var ok bool
	err := s.q.MessageIsBridged.Get(&ok, messageID)
	return ok, err
}

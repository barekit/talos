package consts

const (
	// DefaultDBName is the default database name.
	DefaultDBName = "talos"

	// TableNameMessages is the default table/collection name for messages.
	TableNameMessages = "messages"

	// Column names
	ColSessionID  = "session_id"
	ColRole       = "role"
	ColContent    = "content"
	ColToolCalls  = "tool_calls"
	ColToolCallID = "tool_call_id"
	ColCreatedAt  = "created_at"

	// Neo4j specific
	LabelSession  = "Session"
	LabelMessage  = "Message"
	RelHasMessage = "HAS_MESSAGE"
)

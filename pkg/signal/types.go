package signal

// SignalMessage represents the top-level JSON object from signal-cli-rest-api websocket
type SignalMessage struct {
	Account  string   `json:"account"`
	Envelope Envelope `json:"envelope"`
}

type Envelope struct {
	Source       string       `json:"source"`
	SourceNumber string       `json:"sourceNumber"`
	SourceUuid   string       `json:"sourceUuid"`
	Timestamp    int64        `json:"timestamp"`
	DataMessage  *DataMessage `json:"dataMessage,omitempty"`
	// We capture sync messages too if we want to see what WE sent from other devices
	SyncMessage *SyncMessage `json:"syncMessage,omitempty"`
}

type DataMessage struct {
	Timestamp        int64        `json:"timestamp"`
	Message          string       `json:"message"`
	ExpiresInSeconds int          `json:"expiresInSeconds"`
	ViewOnce         bool         `json:"viewOnce"`
	GroupInfo        *GroupInfo   `json:"groupInfo,omitempty"`
	Attachments      []Attachment `json:"attachments,omitempty"`
}

type SyncMessage struct {
	SentMessage *DataMessage `json:"sentMessage,omitempty"`
}

type GroupInfo struct {
	GroupId string `json:"groupId"`
	Type    string `json:"type"`
}

type Attachment struct {
	ContentType string `json:"contentType"`
	Id          string `json:"id"`
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
}

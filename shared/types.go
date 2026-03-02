package shared

type MessageType int

const (
	PaintMessage MessageType = iota
	ChatMessage
	GameMessage
	NewGameMessage
)

type Message struct {
	From        int         `json:"from"`
	MessageType MessageType `json:"messageType"`
	Content     string      `json:"content"`
	Dest        int         `json:"dest"`
}

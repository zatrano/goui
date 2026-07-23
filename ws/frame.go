package ws

import "encoding/json"

const (
	FrameTypeEvent    = "event"
	FrameTypeRender   = "render"
	FrameTypePush     = "push"
	FrameTypeError    = "error"
	FrameTypeSession  = "session"
	FrameTypePrefetch = "prefetch"
	FrameTypeActivate = "activate"
)

// MaxPrefetch is the per-session cap for silently mounted (not yet visible) components.
const MaxPrefetch = 5

// Frame is the JSON message format exchanged between client and server.
type Frame struct {
	Type      string          `json:"type"`
	Component string          `json:"component,omitempty"`
	Event     string          `json:"event,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// PushMessage is sent to clients as a toast or notification.
type PushMessage struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// SessionPayload carries the session identifier to the client.
type SessionPayload struct {
	ID string `json:"id"`
}

// ErrorPayload carries an error message to the client.
type ErrorPayload struct {
	Message string `json:"message"`
}

func newPushFrame(msg PushMessage) Frame {
	payload, _ := json.Marshal(msg)
	return Frame{
		Type:    FrameTypePush,
		Payload: payload,
	}
}

func newErrorFrame(message string) Frame {
	payload, _ := json.Marshal(ErrorPayload{Message: message})
	return Frame{
		Type:    FrameTypeError,
		Payload: payload,
	}
}

func newSessionFrame(sessionID string) Frame {
	payload, _ := json.Marshal(SessionPayload{ID: sessionID})
	return Frame{
		Type:    FrameTypeSession,
		Payload: payload,
	}
}

func decodeEventPayload(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

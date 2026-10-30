package voicegateway

import (
	"strconv"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/ws"
)

//go:generate go run ../../utils/cmd/genevent -p voicegateway -o event_methods.go

// OpUnmarshalers contains the Op unmarshalers for the voice gateway events.
var OpUnmarshalers = ws.NewOpUnmarshalers()

// IdentifyCommand is a command for Op 0.
//
// https://discord.com/developers/docs/topics/voice-connections#establishing-a-voice-websocket-connection-example-voice-identify-payload
type IdentifyCommand struct {
	GuildID   discord.GuildID `json:"server_id"` // yes, this should be "server_id"
	UserID    discord.UserID  `json:"user_id"`
	SessionID string          `json:"session_id"`
	Token     string          `json:"token"`
	// MaxDAVEProtocolVersion is the highest DAVE protocol version the client
	// supports. 0 means DAVE isn't supported.
	MaxDAVEProtocolVersion int `json:"max_dave_protocol_version"`
}

// SelectProtocolCommand is a command for Op 1.
//
// https://discord.com/developers/docs/topics/voice-connections#establishing-a-voice-udp-connection-example-select-protocol-payload
type SelectProtocolCommand struct {
	Protocol string             `json:"protocol"`
	Data     SelectProtocolData `json:"data"`
}

// SelectProtocolData is the data inside a SelectProtocolCommand.
type SelectProtocolData struct {
	Address string `json:"address"`
	Port    uint16 `json:"port"`
	Mode    string `json:"mode"`
}

// ReadyEvent is an event for Op 2.
//
// https://discord.com/developers/docs/topics/voice-connections#establishing-a-voice-websocket-connection-example-voice-ready-payload
type ReadyEvent struct {
	SSRC        uint32   `json:"ssrc"`
	IP          string   `json:"ip"`
	Port        int      `json:"port"`
	Modes       []string `json:"modes"`
	Experiments []string `json:"experiments"`

	// From Discord's API Docs:
	//
	// `heartbeat_interval` here is an erroneous field and should be ignored.
	// The correct `heartbeat_interval` value comes from the Hello payload.

	// HeartbeatInterval discord.Milliseconds `json:"heartbeat_interval"`
}

// Addr formats the URL inside Ready to be of format "host:port".
func (r ReadyEvent) Addr() string {
	return r.IP + ":" + strconv.Itoa(r.Port)
}

// HeartbeatCommand is a command for Op 3.
//
// https://discord.com/developers/docs/topics/voice-connections#heartbeating-example-heartbeat-payload-since-v8
type HeartbeatCommand struct {
	Nonce uint64 `json:"t"`
	// SeqAck is the sequence number of the last numbered message received
	// from the gateway.
	SeqAck int64 `json:"seq_ack"`
}

// SessionDescriptionEvent is an event for Op 4.
//
// https://discord.com/developers/docs/topics/voice-connections#transport-encryption-modes-example-session-description-payload
type SessionDescriptionEvent struct {
	Mode      string   `json:"mode"`
	SecretKey [32]byte `json:"secret_key"`
	// DAVEProtocolVersion is the DAVE protocol version the call starts with. 0
	// means the call isn't end-to-end encrypted yet.
	DAVEProtocolVersion uint16 `json:"dave_protocol_version"`
}

// https://discord.com/developers/docs/topics/voice-connections#speaking
type SpeakingFlag uint64

const NotSpeaking SpeakingFlag = 0

const (
	Microphone SpeakingFlag = 1 << iota
	Soundshare
	Priority
)

// SpeakingEvent is an event for Op 5. It is also a command.
//
// https://discord.com/developers/docs/topics/voice-connections#speaking-example-speaking-payload
type SpeakingEvent struct {
	Speaking SpeakingFlag   `json:"speaking"`
	Delay    int            `json:"delay"`
	SSRC     uint32         `json:"ssrc"`
	UserID   discord.UserID `json:"user_id,omitempty"`
}

// HeartbeatAckEvent is an event for Op 6.
//
// https://discord.com/developers/docs/topics/voice-connections#heartbeating-example-heartbeat-ack-payload-since-v8
type HeartbeatAckEvent struct {
	Nonce uint64 `json:"t"`
}

// ResumeCommand is a command for Op 7.
//
// https://discord.com/developers/docs/topics/voice-connections#resuming-voice-connection-example-resume-connection-payload-since-v8
type ResumeCommand struct {
	GuildID   discord.GuildID `json:"server_id"` // yes, this should be "server_id"
	SessionID string          `json:"session_id"`
	Token     string          `json:"token"`
	// SeqAck is the sequence number of the last numbered message received
	// from the gateway.
	SeqAck int64 `json:"seq_ack"`
}

// HelloEvent is an event for Op 8.
//
// https://discord.com/developers/docs/topics/voice-connections#heartbeating-example-hello-payload
type HelloEvent struct {
	HeartbeatInterval discord.Milliseconds `json:"heartbeat_interval"`
}

// ResumedEvent is an event for Op 9.
// https://discord.com/developers/docs/topics/voice-connections#resuming-voice-connection-example-resumed-payload
type ResumedEvent struct{}

// ClientsConnectEvent is an event for Op 11. It lists the users that are
// connected to the call.
type ClientsConnectEvent struct {
	UserIDs []discord.UserID `json:"user_ids"`
}

// ClientConnectEvent is an event for Op 12. It is undocumented.
type ClientConnectEvent struct {
	UserID    discord.UserID `json:"user_id"`
	AudioSSRC uint32         `json:"audio_ssrc"`
	VideoSSRC uint32         `json:"video_ssrc"`
}

// ClientDisconnectEvent is an event for Op 13. It is undocumented, but its
// existence is mentioned in this issue:
// https://github.com/discord/discord-api-docs/issues/510.
type ClientDisconnectEvent struct {
	UserID discord.UserID `json:"user_id"`
}

// DAVEPrepareTransitionEvent is an event for Op 21. A downgrade from DAVE or
// a change of its protocol version is coming up.
//
// https://discord.com/developers/docs/topics/voice-connections#end-to-end-encryption-dave-protocol
type DAVEPrepareTransitionEvent struct {
	ProtocolVersion uint16 `json:"protocol_version"`
	TransitionID    uint16 `json:"transition_id"`
}

// DAVEExecuteTransitionEvent is an event for Op 22. The transition with the
// given ID is to be executed now.
type DAVEExecuteTransitionEvent struct {
	TransitionID uint16 `json:"transition_id"`
}

// DAVETransitionReadyCommand is a command for Op 23. It tells the gateway
// that the client is ready for the transition with the given ID.
type DAVETransitionReadyCommand struct {
	TransitionID uint16 `json:"transition_id"`
}

// DAVEPrepareEpochEvent is an event for Op 24. A new MLS group is about to
// be created for the given protocol version.
type DAVEPrepareEpochEvent struct {
	ProtocolVersion uint16 `json:"protocol_version"`
	Epoch           int    `json:"epoch"`
}

// DAVEMLSInvalidCommitWelcomeCommand is a command for Op 31. It tells the
// gateway that a commit or welcome couldn't be processed.
type DAVEMLSInvalidCommitWelcomeCommand struct {
	TransitionID uint16 `json:"transition_id"`
}

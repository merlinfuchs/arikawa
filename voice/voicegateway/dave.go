package voicegateway

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/diamondburned/arikawa/v3/utils/ws"
)

// The DAVE MLS messages are sent as binary websocket messages instead of JSON.
// Messages from the gateway start with a big-endian uint16 sequence number and
// the opcode, messages to the gateway only with the opcode.
//
// https://discord.com/developers/docs/topics/voice-connections#end-to-end-encryption-dave-protocol-binary-websocket-messages

// DAVEMLSExternalSenderPackageEvent is the binary event for Op 25. It holds
// the credential and public key of the MLS external sender.
type DAVEMLSExternalSenderPackageEvent struct {
	ExternalSenderPackage []byte
}

// DAVEMLSKeyPackageCommand is the binary command for Op 26. It holds the MLS
// key package of the client.
type DAVEMLSKeyPackageCommand struct {
	KeyPackage []byte
}

// DAVEMLSProposalsEvent is the binary event for Op 27. It holds MLS proposals
// to be appended or revoked.
type DAVEMLSProposalsEvent struct {
	Proposals []byte
}

// DAVEMLSCommitWelcomeCommand is the binary command for Op 28. It holds an
// MLS commit with an optional MLS welcome.
type DAVEMLSCommitWelcomeCommand struct {
	CommitWelcome []byte
}

// DAVEMLSAnnounceCommitTransitionEvent is the binary event for Op 29. It holds
// the MLS commit to be processed for the upcoming transition.
type DAVEMLSAnnounceCommitTransitionEvent struct {
	TransitionID uint16
	Commit       []byte
}

// DAVEMLSWelcomeEvent is the binary event for Op 30. It holds the MLS welcome
// to the group for the upcoming transition.
type DAVEMLSWelcomeEvent struct {
	TransitionID uint16
	Welcome      []byte
}

// Op implements Event. It always returns Op 25.
func (*DAVEMLSExternalSenderPackageEvent) Op() ws.OpCode { return 25 }

// EventType implements Event.
func (*DAVEMLSExternalSenderPackageEvent) EventType() ws.EventType { return "" }

// Op implements Event. It always returns Op 26.
func (*DAVEMLSKeyPackageCommand) Op() ws.OpCode { return 26 }

// EventType implements Event.
func (*DAVEMLSKeyPackageCommand) EventType() ws.EventType { return "" }

// Op implements Event. It always returns Op 27.
func (*DAVEMLSProposalsEvent) Op() ws.OpCode { return 27 }

// EventType implements Event.
func (*DAVEMLSProposalsEvent) EventType() ws.EventType { return "" }

// Op implements Event. It always returns Op 28.
func (*DAVEMLSCommitWelcomeCommand) Op() ws.OpCode { return 28 }

// EventType implements Event.
func (*DAVEMLSCommitWelcomeCommand) EventType() ws.EventType { return "" }

// Op implements Event. It always returns Op 29.
func (*DAVEMLSAnnounceCommitTransitionEvent) Op() ws.OpCode { return 29 }

// EventType implements Event.
func (*DAVEMLSAnnounceCommitTransitionEvent) EventType() ws.EventType { return "" }

// Op implements Event. It always returns Op 30.
func (*DAVEMLSWelcomeEvent) Op() ws.OpCode { return 30 }

// EventType implements Event.
func (*DAVEMLSWelcomeEvent) EventType() ws.EventType { return "" }

// ErrBinaryMessageTooShort is returned when a binary message is too short to
// hold its header.
var ErrBinaryMessageTooShort = errors.New("binary message too short")

// UnmarshalBinaryOp decodes a binary message from the voice gateway.
func UnmarshalBinaryOp(r io.Reader) (ws.Op, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return ws.Op{}, fmt.Errorf("failed to read binary message: %w", err)
	}

	if len(b) < 3 {
		return ws.Op{}, ErrBinaryMessageTooShort
	}

	seq := binary.BigEndian.Uint16(b[0:2])
	code := ws.OpCode(b[2])
	payload := b[3:]

	var ev ws.Event

	switch code {
	case 25:
		ev = &DAVEMLSExternalSenderPackageEvent{ExternalSenderPackage: payload}
	case 27:
		ev = &DAVEMLSProposalsEvent{Proposals: payload}
	case 29, 30:
		if len(payload) < 2 {
			return ws.Op{}, ErrBinaryMessageTooShort
		}
		transitionID := binary.BigEndian.Uint16(payload[0:2])
		if code == 29 {
			ev = &DAVEMLSAnnounceCommitTransitionEvent{TransitionID: transitionID, Commit: payload[2:]}
		} else {
			ev = &DAVEMLSWelcomeEvent{TransitionID: transitionID, Welcome: payload[2:]}
		}
	default:
		return ws.Op{}, ws.UnknownEventError{Op: code}
	}

	return ws.Op{
		Code:     code,
		Data:     ev,
		Sequence: int64(seq),
	}, nil
}

// marshalBinaryCommand encodes a binary command to the voice gateway.
func marshalBinaryCommand(code ws.OpCode, payload []byte) []byte {
	b := make([]byte, 0, 1+len(payload))
	b = append(b, byte(code))
	return append(b, payload...)
}

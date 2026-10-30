//
// For the brave souls who get this far: You are the chosen ones,
// the valiant knights of programming who toil away, without rest,
// fixing our most awful code.  To you, true saviors, kings of men,
// I say this: never gonna give you up, never gonna let you down,
// never gonna run around and desert you.  Never gonna make you cry,
// never gonna say goodbye.  Never gonna tell a lie and hurt you.
//

package voicegateway

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/disgoorg/godave"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/ws"
)

// Version represents the current version of the Discord Gateway Gateway this package uses.
const Version = "8"

var (
	ErrNoSessionID = errors.New("no sessionID was received")
	ErrNoEndpoint  = errors.New("no endpoint was received")
)

// State contains state information of a voice gateway.
type State struct {
	UserID    discord.UserID  // constant
	GuildID   discord.GuildID // constant
	ChannelID discord.ChannelID

	SessionID string
	Token     string
	Endpoint  string
}

// Gateway represents a Discord Gateway Gateway connection.
type Gateway struct {
	gateway *ws.Gateway
	state   State // constant
	dave    godave.Session

	// seq is the sequence number of the last numbered message received, sent
	// back to the gateway in heartbeats and resumes.
	seq atomic.Int64

	mutex     sync.RWMutex
	ready     *ReadyEvent
	ssrcUsers map[uint32]discord.UserID
}

// DefaultGatewayOpts contains the default options to be used for connecting to
// the gateway.
var DefaultGatewayOpts = ws.GatewayOpts{
	ReconnectDelay: func(try int) time.Duration {
		// minimum 4 seconds
		return time.Duration(4+(2*try)) * time.Second
	},
	// FatalCloseCodes contains the default gateway close codes that will cause
	// the gateway to exit. In other words, it's a list of unrecoverable close
	// codes.
	FatalCloseCodes: []int{
		4003, // not authenticated
		4004, // authentication failed
		4006, // session invalid
		4009, // session timed out
		4011, // server not found
		4012, // unknown protocol
		4014, // disconnected
		4016, // unknown encryption mode
		4017, // E2EE/DAVE protocol required
	},
	DialTimeout:           0,
	ReconnectAttempt:      0,
	AlwaysCloseGracefully: true,
}

// New creates a new voice gateway without end-to-end encryption. Discord
// rejects voice connections without DAVE support, so this is mostly useful
// for testing. Use NewWithDAVE instead.
func New(state State) *Gateway {
	return NewWithDAVE(state, godave.NewNoopSession, slog.New(slog.DiscardHandler))
}

// NewWithDAVE creates a new voice gateway that uses a DAVE session created
// with createDAVE for the end-to-end encryption of the call. The session is
// closed once the gateway is closed.
func NewWithDAVE(state State, createDAVE godave.SessionCreateFunc, logger *slog.Logger) *Gateway {
	// https://discord.com/developers/docs/topics/voice-connections#establishing-a-voice-websocket-connection
	endpoint := "wss://" + strings.TrimSuffix(state.Endpoint, ":80") + "/?v=" + Version

	codec := ws.NewCodec(OpUnmarshalers)
	codec.BinaryUnmarshaler = UnmarshalBinaryOp

	gw := ws.NewGateway(
		ws.NewWebsocket(codec, endpoint),
		&DefaultGatewayOpts,
	)

	g := &Gateway{
		gateway:   gw,
		state:     state,
		ssrcUsers: make(map[uint32]discord.UserID),
	}
	g.dave = createDAVE(logger, godave.UserID(state.UserID.String()), daveCallbacks{g})
	g.dave.SetChannelID(godave.ChannelID(state.ChannelID))

	return g
}

// Ready returns the ready event.
func (g *Gateway) Ready() *ReadyEvent {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	return g.ready
}

// DAVE returns the DAVE session of the call. It encrypts outgoing and
// decrypts incoming audio frames.
func (g *Gateway) DAVE() godave.Session {
	return g.dave
}

// UserIDBySSRC returns the ID of the user who sends audio with the given SSRC,
// or 0 if it's not known yet. It's known once the user started speaking.
func (g *Gateway) UserIDBySSRC(ssrc uint32) discord.UserID {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	return g.ssrcUsers[ssrc]
}

// LastError returns the last error that the gateway has received. It only
// returns a valid error if the gateway's event loop as exited. If the event
// loop hasn't been started AND stopped, the function will panic.
func (g *Gateway) LastError() error {
	return g.gateway.LastError()
}

// Send is a function to send an Op payload to the Gateway.
func (g *Gateway) Send(ctx context.Context, data ws.Event) error {
	switch data := data.(type) {
	case *DAVEMLSKeyPackageCommand:
		return g.gateway.SendBinary(ctx, marshalBinaryCommand(data.Op(), data.KeyPackage))
	case *DAVEMLSCommitWelcomeCommand:
		return g.gateway.SendBinary(ctx, marshalBinaryCommand(data.Op(), data.CommitWelcome))
	}

	return g.gateway.Send(ctx, data)
}

// Speaking sends a Speaking operation (opcode 5) to the Gateway Gateway.
func (g *Gateway) Speaking(ctx context.Context, flag SpeakingFlag) error {
	g.mutex.RLock()
	ready := g.ready
	g.mutex.RUnlock()

	if ready == nil {
		return errors.New("Speaking called before gateway was ready")
	}

	return g.gateway.Send(ctx, &SpeakingEvent{
		Speaking: flag,
		Delay:    0,
		SSRC:     ready.SSRC,
	})
}

func (g *Gateway) Connect(ctx context.Context) <-chan ws.Op {
	return g.gateway.Connect(ctx, (*gatewayImpl)(g))
}

var (
	// ErrMissingForIdentify is an error when we are missing information to
	// identify.
	ErrMissingForIdentify = errors.New("missing GuildID, UserID, SessionID, or Token for identify")
	// ErrMissingForResume is an error when we are missing information to
	// resume.
	ErrMissingForResume = errors.New("missing GuildID, SessionID, or Token for resuming")
)

// daveSendTimeout is how long the DAVE session's messages to the gateway may
// take to be sent.
const daveSendTimeout = 10 * time.Second

// daveCallbacks sends the messages of the DAVE session to the gateway.
type daveCallbacks struct{ g *Gateway }

var _ godave.Callbacks = daveCallbacks{}

func (c daveCallbacks) send(data ws.Event) error {
	ctx, cancel := context.WithTimeout(context.Background(), daveSendTimeout)
	defer cancel()

	return c.g.Send(ctx, data)
}

func (c daveCallbacks) SendMLSKeyPackage(keyPackage []byte) error {
	return c.send(&DAVEMLSKeyPackageCommand{KeyPackage: keyPackage})
}

func (c daveCallbacks) SendMLSCommitWelcome(commitWelcome []byte) error {
	return c.send(&DAVEMLSCommitWelcomeCommand{CommitWelcome: commitWelcome})
}

func (c daveCallbacks) SendReadyForTransition(transitionID uint16) error {
	return c.send(&DAVETransitionReadyCommand{TransitionID: transitionID})
}

func (c daveCallbacks) SendInvalidCommitWelcome(transitionID uint16) error {
	return c.send(&DAVEMLSInvalidCommitWelcomeCommand{TransitionID: transitionID})
}

type gatewayImpl Gateway

func (g *gatewayImpl) sendIdentify(ctx context.Context) error {
	id := IdentifyCommand{
		GuildID:                g.state.GuildID,
		UserID:                 g.state.UserID,
		SessionID:              g.state.SessionID,
		Token:                  g.state.Token,
		MaxDAVEProtocolVersion: g.dave.MaxSupportedProtocolVersion(),
	}
	if !id.GuildID.IsValid() || !id.UserID.IsValid() || id.SessionID == "" || id.Token == "" {
		return ErrMissingForIdentify
	}

	return g.gateway.Send(ctx, &id)
}

func (g *gatewayImpl) sendResume(ctx context.Context) error {
	if !g.state.GuildID.IsValid() || g.state.SessionID == "" || g.state.Token == "" {
		return ErrMissingForResume
	}

	return g.gateway.Send(ctx, &ResumeCommand{
		GuildID:   g.state.GuildID,
		SessionID: g.state.SessionID,
		Token:     g.state.Token,
		SeqAck:    g.seq.Load(),
	})
}

func (g *gatewayImpl) OnOp(ctx context.Context, op ws.Op) bool {
	if op.Sequence > 0 {
		g.seq.Store(op.Sequence)
	}

	switch data := op.Data.(type) {
	case *HelloEvent:
		g.gateway.ResetHeartbeat(data.HeartbeatInterval.Duration())

		g.mutex.RLock()
		ready := g.ready
		g.mutex.RUnlock()

		// Send Discord either the Identify packet (if it's a fresh
		// connection), or a Resume packet (if it's a dead connection).
		if ready == nil {
			// SessionID is empty, so this is a completely new session.
			if err := g.sendIdentify(ctx); err != nil {
				g.gateway.SendErrorWrap(err, "failed to send identify")
				g.gateway.QueueReconnect()
			}
		} else {
			if err := g.sendResume(ctx); err != nil {
				g.gateway.SendErrorWrap(err, "failed to send resume")
				g.gateway.QueueReconnect()
			}
		}
	case *ReadyEvent:
		g.mutex.Lock()
		g.ready = data
		g.mutex.Unlock()

		g.dave.AssignSsrcToCodec(data.SSRC, godave.CodecOpus)
	case *SessionDescriptionEvent:
		g.dave.OnSelectProtocolAck(data.DAVEProtocolVersion)
	case *SpeakingEvent:
		if data.UserID.IsValid() {
			g.mutex.Lock()
			g.ssrcUsers[data.SSRC] = data.UserID
			g.mutex.Unlock()
		}
	case *ClientsConnectEvent:
		for _, userID := range data.UserIDs {
			g.dave.AddUser(godave.UserID(userID.String()))
		}
	case *ClientDisconnectEvent:
		g.mutex.Lock()
		for ssrc, userID := range g.ssrcUsers {
			if userID == data.UserID {
				delete(g.ssrcUsers, ssrc)
			}
		}
		g.mutex.Unlock()

		g.dave.RemoveUser(godave.UserID(data.UserID.String()))
	case *DAVEPrepareTransitionEvent:
		g.dave.OnDavePrepareTransition(data.TransitionID, data.ProtocolVersion)
	case *DAVEExecuteTransitionEvent:
		g.dave.OnDaveExecuteTransition(data.TransitionID)
	case *DAVEPrepareEpochEvent:
		g.dave.OnDavePrepareEpoch(data.Epoch, data.ProtocolVersion)
	case *DAVEMLSExternalSenderPackageEvent:
		g.dave.OnDaveMLSExternalSenderPackage(data.ExternalSenderPackage)
	case *DAVEMLSProposalsEvent:
		g.dave.OnDaveMLSProposals(data.Proposals)
	case *DAVEMLSAnnounceCommitTransitionEvent:
		g.dave.OnDaveMLSPrepareCommitTransition(data.TransitionID, data.Commit)
	case *DAVEMLSWelcomeEvent:
		g.dave.OnDaveMLSWelcome(data.TransitionID, data.Welcome)
	}

	return true
}

func (g *gatewayImpl) SendHeartbeat(ctx context.Context) {
	heartbeat := HeartbeatCommand{
		Nonce:  uint64(time.Now().UnixNano()),
		SeqAck: g.seq.Load(),
	}
	if err := g.gateway.Send(ctx, &heartbeat); err != nil {
		g.gateway.SendErrorWrap(err, "heartbeat error")
		g.gateway.QueueReconnect()
	}
}

func (g *gatewayImpl) Close() error {
	return g.dave.Close()
}

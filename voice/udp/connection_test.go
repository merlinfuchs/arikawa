package udp

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

func newTestConnection(t *testing.T, conn net.Conn, mode string, secret [32]byte) *Connection {
	t.Helper()

	c := &Connection{
		conn:       conn,
		ssrc:       42,
		frequency:  time.NewTicker(time.Millisecond),
		timeIncr:   960,
		stopFreq:   make(chan struct{}),
		packet:     [12]byte{0: 0x80, 1: 0x78},
		recvBuf:    make([]byte, 1400),
		recvOpus:   make([]byte, 1400),
		recvPacket: &Packet{},
	}
	binary.BigEndian.PutUint32(c.packet[8:12], c.ssrc)

	if err := c.UseSecret(mode, secret); err != nil {
		t.Fatal("failed to use secret:", err)
	}

	t.Cleanup(func() { c.Close() })
	return c
}

func TestConnectionRoundTrip(t *testing.T) {
	secret := [32]byte{1, 2, 3, 4, 5}

	for _, mode := range []string{AEADAES256GCMRTPSize, AEADXChaCha20Poly1305RTPSize} {
		t.Run(mode, func(t *testing.T) {
			a, b := net.Pipe()
			sender := newTestConnection(t, a, mode, secret)
			receiver := newTestConnection(t, b, mode, secret)

			frames := [][]byte{[]byte("first opus frame"), []byte("second")}

			go func() {
				for _, frame := range frames {
					if _, err := sender.Write(frame); err != nil {
						t.Error("failed to write:", err)
						return
					}
				}
			}()

			for i, frame := range frames {
				p, err := receiver.ReadPacket()
				if err != nil {
					t.Fatal("failed to read packet:", err)
				}
				if !bytes.Equal(p.Opus, frame) {
					t.Fatalf("frame %d: got %q, want %q", i, p.Opus, frame)
				}
				if p.Sequence() != uint16(i) {
					t.Fatalf("frame %d: got sequence %d", i, p.Sequence())
				}
				if p.SSRC() != 42 {
					t.Fatalf("frame %d: got SSRC %d", i, p.SSRC())
				}
			}
		})
	}
}

func TestConnectionReadPacketWithExtension(t *testing.T) {
	secret := [32]byte{9}

	a, b := net.Pipe()
	sender := newTestConnection(t, a, AEADAES256GCMRTPSize, secret)
	receiver := newTestConnection(t, b, AEADAES256GCMRTPSize, secret)

	// An RTP header with the extension bit set, followed by the unencrypted
	// extension header for 1 word of extension data.
	header := []byte{0x90, 0x78, 0, 7, 0, 0, 0, 1, 0, 0, 0, 42, 0xBE, 0xDE, 0, 1}
	// The extension data is encrypted together with the payload.
	plaintext := append([]byte{0xAA, 0xBB, 0xCC, 0xDD}, []byte("opus")...)

	nonce := make([]byte, sender.aead.NonceSize())
	nonce[0] = 5

	packet := append([]byte{}, header...)
	packet = sender.aead.Seal(packet, nonce, plaintext, header)
	packet = append(packet, nonce[:nonceSize]...)

	go func() {
		if _, err := a.Write(packet); err != nil {
			t.Error("failed to write:", err)
		}
	}()

	p, err := receiver.ReadPacket()
	if err != nil {
		t.Fatal("failed to read packet:", err)
	}
	if string(p.Opus) != "opus" {
		t.Fatalf("got %q, want %q", p.Opus, "opus")
	}
}

func TestConnectionReadPacketWrongSecret(t *testing.T) {
	a, b := net.Pipe()
	sender := newTestConnection(t, a, AEADAES256GCMRTPSize, [32]byte{1})
	receiver := newTestConnection(t, b, AEADAES256GCMRTPSize, [32]byte{2})

	go sender.Write([]byte("opus"))

	if _, err := receiver.ReadPacket(); err != ErrDecryptionFailed {
		t.Fatalf("got error %v, want ErrDecryptionFailed", err)
	}
}

func TestChooseMode(t *testing.T) {
	tests := []struct {
		modes []string
		want  string
	}{
		{[]string{AEADXChaCha20Poly1305RTPSize, AEADAES256GCMRTPSize}, AEADAES256GCMRTPSize},
		{[]string{"xsalsa20_poly1305", AEADXChaCha20Poly1305RTPSize}, AEADXChaCha20Poly1305RTPSize},
	}
	for _, test := range tests {
		got, err := ChooseMode(test.modes)
		if err != nil {
			t.Fatal("unexpected error:", err)
		}
		if got != test.want {
			t.Errorf("ChooseMode(%v) = %q, want %q", test.modes, got, test.want)
		}
	}

	if _, err := ChooseMode([]string{"xsalsa20_poly1305"}); err == nil {
		t.Error("expected an error for unsupported modes")
	}
}

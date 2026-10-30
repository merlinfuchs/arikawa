package ws

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type binaryTestEvent struct{ Payload []byte }

func (*binaryTestEvent) Op() OpCode           { return 42 }
func (*binaryTestEvent) EventType() EventType { return "" }

func TestConnBinaryMessages(t *testing.T) {
	received := make(chan []byte, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error("failed to upgrade:", err)
			return
		}
		defer conn.Close()

		if err := conn.WriteMessage(websocket.BinaryMessage, []byte{1, 2, 3}); err != nil {
			t.Error("failed to write:", err)
			return
		}

		typ, b, err := conn.ReadMessage()
		if err != nil {
			t.Error("failed to read:", err)
			return
		}
		if typ != websocket.BinaryMessage {
			t.Errorf("got message type %d, want binary", typ)
		}
		received <- b
	}))
	defer srv.Close()

	codec := NewCodec(NewOpUnmarshalers())
	codec.BinaryUnmarshaler = func(r io.Reader) (Op, error) {
		b, err := io.ReadAll(r)
		if err != nil {
			return Op{}, err
		}
		return Op{Code: 42, Data: &binaryTestEvent{Payload: b}}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := NewConn(codec)
	ops, err := conn.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http"))
	if err != nil {
		t.Fatal("failed to dial:", err)
	}
	defer conn.Close(false)

	op, err := ReadOp(ctx, ops)
	if err != nil {
		t.Fatal("failed to read op:", err)
	}
	ev, ok := op.Data.(*binaryTestEvent)
	if !ok {
		t.Fatalf("got event %#v, want a binaryTestEvent", op.Data)
	}
	if !bytes.Equal(ev.Payload, []byte{1, 2, 3}) {
		t.Errorf("got payload %v", ev.Payload)
	}

	if err := conn.SendBinary(ctx, []byte{4, 5}); err != nil {
		t.Fatal("failed to send binary:", err)
	}

	select {
	case b := <-received:
		if !bytes.Equal(b, []byte{4, 5}) {
			t.Errorf("server got %v", b)
		}
	case <-ctx.Done():
		t.Fatal("server didn't receive the binary message")
	}
}

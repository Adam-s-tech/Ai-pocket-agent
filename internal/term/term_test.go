package term

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func lineText(t *Term, i int) string {
	var b strings.Builder
	for _, r := range t.Line(i) {
		b.WriteString(r.Text)
	}
	return b.String()
}

func TestLocalShellThroughEmulator(t *testing.T) {
	if !LocalShell {
		t.Skip("no local shell here")
	}
	sess, err := Local(t.TempDir(), 40, 10)
	if err != nil {
		t.Fatal(err)
	}
	changed := make(chan struct{}, 100)
	tm := New("local", sess, 40, 10, func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	defer tm.Close()
	tm.SendText("printf '\\033[1;31mred\\033[0m %s\\n' done")
	tm.SendKey('\r', 0)
	deadline := time.After(5 * time.Second)
	for {
		for i := 0; i < tm.Lines(); i++ {
			if lineText(tm, i) == "red done" {
				var red bool
				for _, r := range tm.Line(i) {
					red = red || (r.Text == "red" && r.Bold && r.FG != "")
				}
				if !red {
					t.Fatalf("style lost: %+v", tm.Line(i))
				}
				return
			}
		}
		select {
		case <-changed:
		case <-deadline:
			var all []string
			for i := 0; i < tm.Lines(); i++ {
				all = append(all, lineText(tm, i))
			}
			t.Fatalf("no output; screen:\n%s", strings.Join(all, "\n"))
		}
	}
}

func TestRunLocal(t *testing.T) {
	if !LocalShell {
		t.Skip("no local shell here")
	}
	out, err := RunLocal(context.Background(), t.TempDir(), "echo hi; exit 3")
	if out != "hi\n" || err == nil {
		t.Fatalf("%q %v", out, err)
	}
}

func TestKeyRoundTrip(t *testing.T) {
	priv, pub, err := NewKey("pocket-agent")
	if err != nil {
		t.Fatal(err)
	}
	again, err := PublicKey(priv, "pocket-agent")
	if err != nil || again != pub || !strings.HasPrefix(pub, "ssh-ed25519 ") {
		t.Fatalf("%q %q %v", pub, again, err)
	}
}

func TestKnownHosts(t *testing.T) {
	k := &KnownHosts{Path: filepath.Join(t.TempDir(), "known_hosts")}
	key := func() ssh.PublicKey {
		pub, _, _ := ed25519.GenerateKey(rand.Reader)
		p, _ := ssh.NewPublicKey(pub)
		return p
	}
	a, b := key(), key()
	addr := &net.TCPAddr{}
	if err := k.callback(nil)("h:22", addr, a); err == nil {
		t.Fatal("unknown host accepted without asking")
	}
	asked := 0
	trust := func(string, string) bool { asked++; return true }
	if err := k.callback(trust)("h:22", addr, a); err != nil || asked != 1 {
		t.Fatal(err, asked)
	}
	if err := k.callback(nil)("h:22", addr, a); err != nil {
		t.Fatal("trusted host refused:", err)
	}
	if err := k.callback(trust)("h:22", addr, b); !errors.Is(err, ErrHostKeyChanged) {
		t.Fatal("changed key accepted:", err)
	}
	k.Forget("h:22")
	if err := k.callback(trust)("h:22", addr, b); err != nil {
		t.Fatal(err)
	}
}

package term

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"io"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// testServer accepts password "pw" or clientKey, echoes a shell's input
// back upper-cased, and answers the command "hello" with "hi there".
func testServer(t *testing.T, clientKey ssh.PublicKey) (addr string, port int) {
	t.Helper()
	_, hostPriv, _ := ed25519.GenerateKey(rand.Reader)
	signer, _ := ssh.NewSignerFromKey(hostPriv)
	cfg := &ssh.ServerConfig{
		PasswordCallback: func(_ ssh.ConnMetadata, pw []byte) (*ssh.Permissions, error) {
			if string(pw) == "pw" {
				return nil, nil
			}
			return nil, io.EOF
		},
		PublicKeyCallback: func(_ ssh.ConnMetadata, k ssh.PublicKey) (*ssh.Permissions, error) {
			if clientKey != nil && string(k.Marshal()) == string(clientKey.Marshal()) {
				return nil, nil
			}
			return nil, io.EOF
		},
	}
	cfg.AddHostKey(signer)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				_, chans, reqs, err := ssh.NewServerConn(c, cfg)
				if err != nil {
					return
				}
				go ssh.DiscardRequests(reqs)
				for nc := range chans {
					ch, creqs, _ := nc.Accept()
					go func() {
						for r := range creqs {
							switch r.Type {
							case "exec":
								cmd := string(r.Payload[4:])
								r.Reply(true, nil)
								if cmd == "hello" {
									io.WriteString(ch, "hi there\n")
								}
								ch.SendRequest("exit-status", false, binary.BigEndian.AppendUint32(nil, 0))
								ch.Close()
							case "shell":
								r.Reply(true, nil)
								go func() {
									buf := make([]byte, 256)
									for {
										n, err := ch.Read(buf)
										if err != nil {
											return
										}
										s := strings.ReplaceAll(strings.ToUpper(string(buf[:n])), "\r", "\r\n")
										io.WriteString(ch, s)
									}
								}()
							default:
								r.Reply(true, nil) // pty-req, window-change
							}
						}
					}()
				}
			}()
		}
	}()
	a := ln.Addr().(*net.TCPAddr)
	return a.IP.String(), a.Port
}

func TestSSHShellAndCommand(t *testing.T) {
	priv, pub, _ := NewKey("test")
	pk, _, _, _, _ := ssh.ParseAuthorizedKey([]byte(pub))
	addr, port := testServer(t, pk)
	known := &KnownHosts{Path: filepath.Join(t.TempDir(), "known_hosts")}
	ctx := context.Background()

	// Commands need a trusted host.
	if _, err := RunSSH(ctx, Host{Addr: addr, Port: port, User: "u", Password: "pw"}, known, "hello"); err == nil {
		t.Fatal("untrusted host reached")
	}
	trusted := ""
	sess, err := DialSSH(ctx, Host{Addr: addr, Port: port, User: "u", Password: "pw"}, known,
		func(host, fp string) bool { trusted = fp; return true }, 40, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(trusted, "SHA256:") {
		t.Fatalf("fingerprint %q", trusted)
	}
	changed := make(chan struct{}, 100)
	tm := New("ssh", sess, 40, 10, func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	defer tm.Close()
	tm.SendText("echo ok")
	tm.SendKey('\r', 0)
	deadline := time.After(5 * time.Second)
	for found := false; !found; {
		for i := 0; i < tm.Lines(); i++ {
			found = found || lineText(tm, i) == "ECHO OK"
		}
		if found {
			break
		}
		select {
		case <-changed:
		case <-deadline:
			t.Fatalf("no echo; first line %q", lineText(tm, 0))
		}
	}

	// The app's key signs in, and the host is trusted now.
	out, err := RunSSH(ctx, Host{Addr: addr, Port: port, User: "u", Key: priv}, known, "hello")
	if err != nil || out != "hi there\n" {
		t.Fatalf("%q %v", out, err)
	}
}

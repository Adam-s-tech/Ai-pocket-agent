package term

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// Host is where to connect and how to sign in.
type Host struct {
	Addr string // host name or IP
	Port int
	User string
	// Password, or Key (an OpenSSH private key), signs in.
	Password string
	Key      []byte
}

// Address is host:port, as known_hosts records it.
func (h Host) Address() string {
	port := h.Port
	if port == 0 {
		port = 22
	}
	return net.JoinHostPort(h.Addr, strconv.Itoa(port))
}

// TrustFunc decides about a host key seen for the first time, given its
// SHA256 fingerprint; it may block while it asks the user.
type TrustFunc func(host, fingerprint string) bool

// KnownHosts remembers the host keys the user trusted, in a file.
type KnownHosts struct {
	Path string
	mu   sync.Mutex
}

// ErrHostKeyChanged is returned when a host shows a different key than
// the one trusted before - a reinstalled server, or someone in between.
var ErrHostKeyChanged = errors.New("the host's key changed since you last connected; if you expected that, forget the host's key in its settings and connect again")

// callback checks host keys against the file, asking trust about new
// ones; with trust nil, new hosts are refused.
func (k *KnownHosts) callback(trust TrustFunc) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		entry := key.Type() + " " + ssh.FingerprintSHA256(key)
		k.mu.Lock()
		known, ok := k.lookup(hostname)
		k.mu.Unlock()
		switch {
		case ok && known == entry:
			return nil
		case ok:
			return ErrHostKeyChanged
		case trust == nil:
			return fmt.Errorf("%s isn't a trusted host yet: connect to it once from the Terminal to check and trust its key", hostname)
		case !trust(hostname, ssh.FingerprintSHA256(key)):
			return errors.New("the host's key wasn't trusted")
		}
		k.mu.Lock()
		defer k.mu.Unlock()
		f, err := os.OpenFile(k.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = fmt.Fprintf(f, "%s %s\n", hostname, entry)
		return err
	}
}

func (k *KnownHosts) lookup(hostname string) (string, bool) {
	b, err := os.ReadFile(k.Path)
	if err != nil {
		return "", false
	}
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		if h, rest, ok := strings.Cut(sc.Text(), " "); ok && h == hostname {
			return rest, true
		}
	}
	return "", false
}

// Forget drops the trusted key of host (host:port).
func (k *KnownHosts) Forget(hostport string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	b, err := os.ReadFile(k.Path)
	if err != nil {
		return
	}
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if h, _, _ := strings.Cut(l, " "); h != hostport && l != "" {
			out = append(out, l)
		}
	}
	_ = os.WriteFile(k.Path, []byte(strings.Join(out, "\n")+"\n"), 0o600)
}

func (h Host) config(cb ssh.HostKeyCallback) (*ssh.ClientConfig, error) {
	var auth []ssh.AuthMethod
	if len(h.Key) > 0 {
		signer, err := ssh.ParsePrivateKey(h.Key)
		if err != nil {
			return nil, fmt.Errorf("the SSH key can't be read: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}
	if h.Password != "" {
		auth = append(auth, ssh.Password(h.Password),
			ssh.KeyboardInteractive(func(_, _ string, qs []string, _ []bool) ([]string, error) {
				answers := make([]string, len(qs))
				for i := range answers {
					answers[i] = h.Password
				}
				return answers, nil
			}))
	}
	if len(auth) == 0 {
		return nil, errors.New("set a password or use the app's SSH key for this host")
	}
	return &ssh.ClientConfig{User: h.User, Auth: auth, HostKeyCallback: cb, Timeout: 20 * time.Second}, nil
}

func dial(ctx context.Context, h Host, cb ssh.HostKeyCallback) (*ssh.Client, error) {
	cfg, err := h.config(cb)
	if err != nil {
		return nil, err
	}
	d := net.Dialer{Timeout: 20 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", h.Address())
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, h.Address(), cfg)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}

// sshSession is an interactive shell on a host.
type sshSession struct {
	client *ssh.Client
	sess   *ssh.Session
	stdin  io.WriteCloser
	out    *io.PipeReader
}

// DialSSH opens a shell on h in a cols x rows terminal.
func DialSSH(ctx context.Context, h Host, known *KnownHosts, trust TrustFunc, cols, rows int) (Session, error) {
	client, err := dial(ctx, h, known.callback(trust))
	if err != nil {
		return nil, err
	}
	sess, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, err
	}
	modes := ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 115200, ssh.TTY_OP_OSPEED: 115200}
	if err := sess.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		client.Close()
		return nil, err
	}
	stdin, err := sess.StdinPipe()
	if err != nil {
		client.Close()
		return nil, err
	}
	pr, pw := io.Pipe()
	sess.Stdout, sess.Stderr = pw, pw
	if err := sess.Shell(); err != nil {
		client.Close()
		return nil, err
	}
	go func() {
		err := sess.Wait()
		client.Close()
		if err == nil {
			err = io.EOF
		}
		pw.CloseWithError(err)
	}()
	return &sshSession{client: client, sess: sess, stdin: stdin, out: pr}, nil
}

func (s *sshSession) Read(p []byte) (int, error)  { return s.out.Read(p) }
func (s *sshSession) Write(p []byte) (int, error) { return s.stdin.Write(p) }
func (s *sshSession) Resize(cols, rows int) error  { return s.sess.WindowChange(rows, cols) }
func (s *sshSession) Close() error {
	_ = s.sess.Close()
	return s.client.Close()
}

// maxOutput caps a command's output returned to an agent.
const maxOutput = 30000

// RunSSH runs one command on h and returns its combined output, cut to
// maxOutput. Only hosts already trusted are reached.
func RunSSH(ctx context.Context, h Host, known *KnownHosts, command string) (string, error) {
	client, err := dial(ctx, h, known.callback(nil))
	if err != nil {
		return "", err
	}
	defer client.Close()
	stop := context.AfterFunc(ctx, func() { client.Close() })
	defer stop()
	sess, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(command)
	return clip(out), exitErr(err)
}

// exitErr words a command's exit status; output is returned either way.
func exitErr(err error) error {
	var ee *ssh.ExitError
	if errors.As(err, &ee) {
		return fmt.Errorf("exit status %d", ee.ExitStatus())
	}
	return err
}

func clip(b []byte) string {
	s := strings.ToValidUTF8(string(b), "�")
	if r := []rune(s); len(r) > maxOutput {
		return string(r[:maxOutput/2]) + fmt.Sprintf("\n[… %d characters cut …]\n", len(r)-maxOutput) + string(r[len(r)-maxOutput/2:])
	}
	return s
}

// NewKey makes an Ed25519 key pair: the private key in OpenSSH's PEM
// format, and the public key as an authorized_keys line.
func NewKey(comment string) (private []byte, public string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, "", err
	}
	block, err := ssh.MarshalPrivateKey(priv, comment)
	if err != nil {
		return nil, "", err
	}
	sp, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, "", err
	}
	return pem.EncodeToMemory(block), PublicKeyLine(sp, comment), nil
}

// PublicKey returns the authorized_keys line for a private key.
func PublicKey(private []byte, comment string) (string, error) {
	signer, err := ssh.ParsePrivateKey(private)
	if err != nil {
		return "", err
	}
	return PublicKeyLine(signer.PublicKey(), comment), nil
}

// PublicKeyLine formats a public key for authorized_keys.
func PublicKeyLine(k ssh.PublicKey, comment string) string {
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(k))) + " " + comment
}

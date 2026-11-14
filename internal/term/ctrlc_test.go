package term

import (
	"strings"
	"testing"
	"time"
)

func TestCtrlCInterrupts(t *testing.T) {
	if !LocalShell {
		t.Skip()
	}
	sess, err := Local(t.TempDir(), 60, 10)
	if err != nil {
		t.Fatal(err)
	}
	tm := New("local", sess, 60, 10, func() {})
	defer tm.Close()
	time.Sleep(300 * time.Millisecond)
	tm.SendText("sleep 30")
	tm.SendKey('\r', 0)
	time.Sleep(300 * time.Millisecond)
	tm.SendKey('c', 4) // ModCtrl
	time.Sleep(300 * time.Millisecond)
	tm.SendText("echo back")
	tm.SendKey('\r', 0)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for i := 0; i < tm.Lines(); i++ {
			if lineText(tm, i) == "back" {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	var all []string
	for i := 0; i < tm.Lines(); i++ {
		all = append(all, lineText(tm, i))
	}
	t.Fatalf("sleep wasn't interrupted:\n%s", strings.Join(all, "\n"))
}

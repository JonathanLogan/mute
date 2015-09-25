package mixaddr

import (
	"crypto/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/mutecomm/mute/util/times"

	"github.com/agl/ed25519"
)

var testDir = path.Join(os.TempDir(), "testkeys")

func TestKeyList(t *testing.T) {
	now := times.Now()
	timeNow = func() int64 { return now - 2 }
	_, privkey, _ := ed25519.GenerateKey(rand.Reader)
	kl := New(privkey, "mix@mute.berlin", 5, testDir)
	kl.AddKey()
	kl.AddKey()
	timeNow = func() int64 { return now }
	kl.AddKey()
	timeNow = func() int64 { return times.Now() }
	marshalled := kl.Marshal()
	kl2 := New(privkey, "mix@mute.berlin", 5, testDir)
	err := kl2.Unmarshal(marshalled)
	if err != nil {
		t.Errorf("Unmarshal failed: %s", err)
	}
	for k := range kl.Keys {
		if _, ok := kl2.Keys[k]; !ok {
			t.Errorf("Not found after unmarshal: %x", k)
		}
	}
	first, last := kl2.GetBoundaryTime()
	if last-first < 1 {
		t.Error("GetBoundaryTime failed")
	}

	timeNow = func() int64 { return now + 4 }
	// kl2.Maintain()
	kl2.Expire()
	if len(kl2.Keys) != 1 {
		t.Error("Expire wrong number of keys")
	}
	timeNow = func() int64 { return times.Now() + 10 }
	kl2.AddKey()
	kl2.Expire()
	if len(kl2.Keys) != 1 {
		t.Error("Expire wrong number of keys")
	}
	if !testing.Short() {
		kl2 = New(privkey, "mix@mute.berlin", 10, testDir)
		kl2.Maintain()
		time.Sleep(time.Second * 30)
		close(kl2.stopchan)
		time.Sleep(time.Second * 1)
		if len(kl2.Keys) > 4 || len(kl2.Keys) < 2 {
			t.Error("Expire/Add maintainer inconsistent")
		}
	}
}
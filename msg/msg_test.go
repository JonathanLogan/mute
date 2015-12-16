// Copyright (c) 2015 Mute Communications Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package msg

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/agl/ed25519"
	"github.com/mutecomm/mute/cipher"
	"github.com/mutecomm/mute/encode/base64"
	"github.com/mutecomm/mute/keyserver/hashchain"
	"github.com/mutecomm/mute/log"
	"github.com/mutecomm/mute/msg/session/memstore"
	"github.com/mutecomm/mute/uid"
	"github.com/mutecomm/mute/util/fuzzer"
	"github.com/mutecomm/mute/util/msgs"
	"github.com/mutecomm/mute/util/times"
)

func init() {
	if err := log.Init("info", "msg  ", "", true); err != nil {
		panic(err)
	}
}

func encrypt(sign bool, flipUIDs bool) (
	sender, recipient *uid.Message,
	w bytes.Buffer,
	recipientTemp *uid.KeyEntry,
	privateKey string,
	err error,
) {
	sender, err = uid.Create("alice@mute.berlin", false, "", "", uid.Strict,
		hashchain.TestEntry, cipher.RandReader)
	if err != nil {
		return
	}
	recipient, err = uid.Create("bob@mute.berlin", false, "", "", uid.Strict,
		hashchain.TestEntry, cipher.RandReader)
	if err != nil {
		return
	}
	if flipUIDs {
		sender, recipient = recipient, sender
	}
	r := bytes.NewBufferString(msgs.Message1)
	now := uint64(times.Now())
	recipientKI, _, privateKey, err := recipient.KeyInit(1, now+times.Day, now-times.Day,
		false, "mute.berlin", "", "", cipher.RandReader)
	if err != nil {
		return
	}
	recipientTemp, err = recipientKI.KeyEntryECDHE25519(recipient.SigPubKey())
	if err != nil {
		return
	}
	// encrypt
	var privateSigKey *[64]byte
	if sign {
		privateSigKey = sender.PrivateSigKey64()
	}
	ms := memstore.New()
	ms.AddPublicKeyEntry(recipient.Identity(), recipientTemp)
	args := &EncryptArgs{
		Writer: &w,
		From:   sender,
		To:     recipient,
		SenderLastKeychainHash: hashchain.TestEntry,
		PrivateSigKey:          privateSigKey,
		Reader:                 r,
		Rand:                   cipher.RandReader,
		KeyStore:               ms,
	}
	if _, err = Encrypt(args); err != nil {
		return
	}
	return
}

func decrypt(sender, recipient *uid.Message, r io.Reader, recipientTemp *uid.KeyEntry,
	privateKey string, sign bool, chkMsg bool) error {
	// decrypt
	var res bytes.Buffer
	identities := []string{recipient.Identity()}
	recipientIdentities := []*uid.KeyEntry{recipient.PubKey()}
	input := base64.NewDecoder(r)
	version, preHeader, err := ReadFirstOuterHeader(input)
	if err != nil {
		return err
	}
	if version != Version {
		return errors.New("wrong version")
	}
	ms := memstore.New()
	if err := recipientTemp.SetPrivateKey(privateKey); err != nil {
		return err
	}
	ms.AddPrivateKeyEntry(recipientTemp)
	args := &DecryptArgs{
		Writer:              &res,
		Identities:          identities,
		RecipientIdentities: recipientIdentities,
		PreHeader:           preHeader,
		Reader:              input,
		Rand:                cipher.RandReader,
		KeyStore:            ms,
	}
	_, sig, err := Decrypt(args)
	if err != nil {
		return err
	}
	// do not compare messages when fuzzing, because messages have to be different!
	if chkMsg && res.String() != msgs.Message1 {
		return errors.New("messages differ")
	}
	if sign {
		contentHash := cipher.SHA512(res.Bytes())
		decSig, err := base64.Decode(sig)
		if err != nil {
			return err
		}
		if len(decSig) != ed25519.SignatureSize {
			return errors.New("signature has wrong length")
		}
		var sigBuf [ed25519.SignatureSize]byte
		copy(sigBuf[:], decSig)
		if !ed25519.Verify(sender.PublicSigKey32(), contentHash, &sigBuf) {
			return errors.New("signature verification failed")
		}
	}
	return nil
}

func encryptAndDecrypt(t *testing.T, sign, flipUIDs bool) {
	// encrypt
	sender, recipient, w, recipientTemp, privateKey, err := encrypt(sign, flipUIDs)
	if err != nil {
		t.Fatal(err)
	}
	// decrypt
	err = decrypt(sender, recipient, &w, recipientTemp, privateKey, sign, true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUnsignedMsg(t *testing.T) {
	t.Parallel()
	encryptAndDecrypt(t, false, false)
}

func TestSignedMsg(t *testing.T) {
	t.Parallel()
	encryptAndDecrypt(t, true, false)
}

func TestUnsignedMsgFlip(t *testing.T) {
	t.Parallel()
	encryptAndDecrypt(t, false, true)
}

func TestSignedMsgFlip(t *testing.T) {
	t.Parallel()
	encryptAndDecrypt(t, true, true)
}

func encryptAndDecryptFuzzing(t *testing.T, sign bool) {
	// encrypt
	sender, recipient, w, recipientTemp, privateKey, err := encrypt(sign, false)
	if err != nil {
		t.Fatal(err)
	}

	// decrypt func
	testFunc := func(b []byte) error {
		err := decrypt(sender, recipient, bytes.NewBuffer(b), recipientTemp, privateKey, sign, false)
		if err != nil {
			return err
		}
		return nil
	}

	// check length of encrypted message
	if w.Len() != EncodedMsgSize {
		t.Errorf("w.Len() = %d != %d = EncodedMsgSize)",
			w.Len(), EncodedMsgSize)
	}

	// fuzzer: fuzz everything except for most of the message padding
	fzzr := &fuzzer.SequentialFuzzer{
		Data:     w.Bytes(),
		End:      8000 * 8,
		TestFunc: testFunc,
	}
	if ok := fzzr.Fuzz(); !ok {
		t.Error("fuzzer failed")
	}

	fzzr = &fuzzer.SequentialFuzzer{
		Data:     w.Bytes(),
		Start:    EncodedMsgSize*8 - 1000,
		TestFunc: testFunc,
	}
	if ok := fzzr.Fuzz(); !ok {
		t.Error("fuzzer failed")
	}
}

func TestFuzzedUnsignedMsg(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	encryptAndDecryptFuzzing(t, false)
}

func TestFuzzedSignedMsg(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	encryptAndDecryptFuzzing(t, true)
}

func TestUnencodedMsgSize(t *testing.T) {
	t.Parallel()
	if UnencodedMsgSize != 49152 {
		t.Errorf("unencodedMsgSize = %d != %d", UnencodedMsgSize, 49152)
	}
}

func TestMaxContentLength(t *testing.T) {
	t.Parallel()
	if MaxContentLength != 41691 {
		t.Errorf("MaxContentLength = %d != %d", MaxContentLength, 41691)
	}
}

func TestReflection(t *testing.T) {
	alice := "alice@mute.berlin"
	aliceUID, err := uid.Create(alice, false, "", "", uid.Strict,
		hashchain.TestEntry, cipher.RandReader)
	if err != nil {
		t.Fatal(err)
	}
	bob := "bob@mute.berlin"
	bobUID, err := uid.Create(bob, false, "", "", uid.Strict,
		hashchain.TestEntry, cipher.RandReader)
	if err != nil {
		t.Fatal(err)
	}
	var encMsg bytes.Buffer
	aliceKeyStore := memstore.New()
	aliceKeyStore.AddPublicKeyEntry(bob, bobUID.PubKey()) // duplicate key
	encryptArgs := &EncryptArgs{
		Writer: &encMsg,
		From:   aliceUID,
		To:     bobUID,
		SenderLastKeychainHash: hashchain.TestEntry,
		Reader:                 bytes.NewBufferString(msgs.Message1),
		Rand:                   cipher.RandReader,
		KeyStore:               aliceKeyStore,
	}
	if _, err = Encrypt(encryptArgs); err != ErrReflection {
		t.Error("should fail with ErrReflection")
	}
}

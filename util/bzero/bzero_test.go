// Copyright (c) 2015 Mute Communications Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bzero

import (
	"bytes"
	"io"
	"testing"

	"github.com/mutecomm/mute/cipher"
)

func TestBytes(t *testing.T) {
	zero := make([]byte, 1024)
	buf := make([]byte, 1024)
	// compare new buffer
	if !bytes.Equal(buf, zero) {
		t.Error("buffers differ")
	}
	// fill buffer with random data
	if _, err := io.ReadFull(cipher.RandReader, buf); err != nil {
		t.Fatal(err)
	}
	// zero
	Bytes(buf)
	// compare reset buffer
	if !bytes.Equal(buf, zero) {
		t.Error("buffers differ")
	}
}

// Copyright (c) 2015 Mute Communications Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package msg

import (
	"github.com/mutecomm/mute/msg/session"
	"github.com/mutecomm/mute/uid"
	"github.com/mutecomm/mute/util/times"
)

func addSessionKey(ss session.Store, ke *uid.KeyEntry) error {
	ct := uint64(times.Now()) + CleanupTime
	return ss.AddSessionKey(ke.HASH, string(ke.JSON()), ke.PrivateKey(), ct)
}

func getSessionKey(ss session.Store, hash string) (*uid.KeyEntry, error) {
	jsn, privKey, err := ss.GetSessionKey(hash)
	if err != nil {
		return nil, err
	}
	ke, err := uid.NewJSONKeyEntry([]byte(jsn))
	if err != nil {
		return nil, err
	}
	if err := ke.SetPrivateKey(privKey); err != nil {
		return nil, err
	}
	return ke, err
}

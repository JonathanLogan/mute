package msg

import (
	"crypto/sha512"
	"io"

	"github.com/mutecomm/mute/cipher"
	"github.com/mutecomm/mute/encode/base64"
	"github.com/mutecomm/mute/util/bzero"
	"golang.org/x/crypto/hkdf"
)

func symmetricKeys(messageKey []byte) (cryptoKey []byte, hmacKey []byte, err error) {
	// TODO: correct KDF?
	hkdf := hkdf.New(sha512.New, messageKey, nil, nil)

	// derive crypto key for AES-256
	cryptoKey = make([]byte, 32)
	if _, err := io.ReadFull(hkdf, cryptoKey); err != nil {
		return nil, nil, err
	}

	// derive HMAC key for SHA-512 HMAC (TODO: correct size?)
	hmacKey = make([]byte, 64)
	if _, err := io.ReadFull(hkdf, hmacKey); err != nil {
		return nil, nil, err
	}

	return
}

func deriveRootKey(t1, t2, t3 *[32]byte, previousRootKeyHash []byte) ([]byte, error) {
	master := make([]byte, 96+len(previousRootKeyHash))
	copy(master[:], t1[:])
	copy(master[32:], t2[:])
	copy(master[64:], t3[:])
	if previousRootKeyHash != nil {
		copy(master[96:], previousRootKeyHash)
	}

	// TODO: correct KDF?
	hkdf := hkdf.New(sha512.New, master, nil, nil)

	// derive root key
	rootKey := make([]byte, 24)
	if _, err := io.ReadFull(hkdf, rootKey); err != nil {
		return nil, err
	}

	return rootKey, nil
}

func generateMessageKeys(senderIdentity, recipientIdentity string,
	rootKey, senderSessionPub, recipientPub []byte,
	storeSession StoreSession) ([]byte, error) {
	var (
		identities string
		send       []string
		recv       []string
		messageKey []byte
	)

	// identity_fix = HASH(SORT(SenderNym, RecipientNym))
	if senderIdentity < recipientIdentity {
		identities = senderIdentity + recipientIdentity
	} else {
		identities = recipientIdentity + senderIdentity
	}
	identityFix := cipher.SHA512([]byte(identities))

	chainKey := rootKey
	for i := 0; i < NumOfFutureKeys; i++ {
		// messagekey_send[i] = HMAC_HASH(chainkey, "MESSAGE" | HASH(RecipientPub) | identity_fix)
		buffer := append([]byte("MESSAGE"), cipher.SHA512(recipientPub)...)
		buffer = append(buffer, identityFix...)
		send = append(send, base64.Encode(cipher.HMAC(chainKey, buffer)))

		// messagekey_recv[i] = HMAC_HASH(chainkey, "MESSAGE" | HASH(SenderSessionPub) | identity_fix)
		buffer = append([]byte("MESSAGE"), cipher.SHA512(senderSessionPub)...)
		buffer = append(buffer, identityFix...)
		recv = append(recv, base64.Encode(cipher.HMAC(chainKey, buffer)))

		// chainkey = HMAC_HASH(chainkey, "CHAIN" )
		chainKey = cipher.HMAC(chainKey, []byte("CHAIN"))
	}

	// calculate root key hash
	rootKeyHash := base64.Encode(cipher.SHA512(rootKey))
	bzero.Bytes(rootKey)

	// store session
	err := storeSession(senderIdentity, recipientIdentity, rootKeyHash,
		base64.Encode(chainKey), send, recv)
	if err != nil {
		return nil, err
	}

	return messageKey, nil
}
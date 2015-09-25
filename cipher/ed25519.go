package cipher

import (
	"io"

	"github.com/agl/ed25519"
	"github.com/mutecomm/mute/log"
)

// Ed25519Key holds a Ed25519 key pair.
type Ed25519Key struct {
	publicKey  *[ed25519.PublicKeySize]byte
	privateKey *[ed25519.PrivateKeySize]byte
}

// Ed25519Generate generates a new Ed25519 key pair.
func Ed25519Generate(rand io.Reader) (*Ed25519Key, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand)
	if err != nil {
		return nil, err
	}
	return &Ed25519Key{publicKey, privateKey}, nil
}

// PublicKey returns the public key of an ed25519Key.
func (ed25519Key *Ed25519Key) PublicKey() *[32]byte {
	return ed25519Key.publicKey
}

// PrivateKey returns the private key of an ed25519Key.
func (ed25519Key *Ed25519Key) PrivateKey() *[64]byte {
	return ed25519Key.privateKey
}

// SetPublicKey sets the public key of ed25519Key to key.
// SetPublicKey returns an error, if len(key) != ed25519.PublicKeySize.
func (ed25519Key *Ed25519Key) SetPublicKey(key []byte) error {
	if len(key) != ed25519.PublicKeySize {
		return log.Errorf("cipher: Ed25519Key.SetPublicKey(): len(key) = %d != %d = ed25519.PublicKeySize",
			len(key), ed25519.PublicKeySize)
	}
	ed25519Key.publicKey = new([ed25519.PublicKeySize]byte)
	copy(ed25519Key.publicKey[:], key)
	return nil
}

// SetPrivateKey sets the private key of ed25519Key to key.
// SetPrivateKey returns an error, if len(key) != ed25519.PrivateKeySize.
func (ed25519Key *Ed25519Key) SetPrivateKey(key []byte) error {
	if len(key) != ed25519.PrivateKeySize {
		return log.Errorf("cipher: Ed25519Key.SetPrivateKey(): len(key) = %d != %d = ed25519.PrivateKeySize",
			len(key), ed25519.PrivateKeySize)
	}
	ed25519Key.privateKey = new([ed25519.PrivateKeySize]byte)
	copy(ed25519Key.privateKey[:], key)
	return nil
}

// Sign signs the given message with ed25519Key and returns the signature.
func (ed25519Key *Ed25519Key) Sign(message []byte) []byte {
	sig := ed25519.Sign(ed25519Key.privateKey, message)
	return sig[:]
}

// Verify verifies that the signature sig for message is valid for ed25519Key.
func (ed25519Key *Ed25519Key) Verify(message []byte, sig []byte) bool {
	var signature [ed25519.SignatureSize]byte
	copy(signature[:], sig)
	return ed25519.Verify(ed25519Key.publicKey, message, &signature)
}

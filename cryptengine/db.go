package cryptengine

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/mutecomm/mute/keydb"
	"github.com/mutecomm/mute/log"
	"github.com/mutecomm/mute/util/bzero"
)

// create a new KeyDB.
func (ce *CryptEngine) create(homedir string, iterations, passphraseFD int) error {
	keydbname := path.Join(homedir, "keys")
	// read passphrase
	log.Infof("read passphrase from fd %d", passphraseFD)
	fp := os.NewFile(uintptr(passphraseFD), "passphrase-fd")
	scanner := bufio.NewScanner(fp)
	var passphrase []byte
	defer bzero.Bytes(passphrase)
	if scanner.Scan() {
		passphrase = scanner.Bytes()
	} else if err := scanner.Err(); err != nil {
		return log.Error(err)
	}
	// read passphrase again
	log.Infof("read passphrase from fd %d again", passphraseFD)
	var passphrase2 []byte
	defer bzero.Bytes(passphrase2)
	if scanner.Scan() {
		passphrase2 = scanner.Bytes()
	} else if err := scanner.Err(); err != nil {
		return log.Error(err)
	}
	// compare passphrases
	if !bytes.Equal(passphrase, passphrase2) {
		return log.Error("passphrases differ")
	}
	// create keyDB
	log.Infof("create keyDB '%s'", keydbname)
	if err := keydb.Create(keydbname, passphrase, iterations); err != nil {
		return err
	}
	return nil
}

// rekey a KeyDB.
func (ce *CryptEngine) rekey(homedir string, iterations, passphraseFD int) error {
	keydbname := path.Join(homedir, "keys")
	// read old passphrase
	log.Infof("read old passphrase from fd %d", passphraseFD)
	fp := os.NewFile(uintptr(passphraseFD), "passphrase-fd")
	scanner := bufio.NewScanner(fp)
	var oldPassphrase []byte
	defer bzero.Bytes(oldPassphrase)
	if scanner.Scan() {
		oldPassphrase = scanner.Bytes()
	} else if err := scanner.Err(); err != nil {
		return log.Error(err)
	}
	// read new passphrase
	log.Infof("read new passphrase from fd %d", passphraseFD)
	var newPassphrase []byte
	defer bzero.Bytes(newPassphrase)
	if scanner.Scan() {
		newPassphrase = scanner.Bytes()
	} else if err := scanner.Err(); err != nil {
		return log.Error(err)
	}
	// read new passphrase again
	log.Infof("read new passphrase from fd %d again", passphraseFD)
	var newPassphrase2 []byte
	defer bzero.Bytes(newPassphrase2)
	if scanner.Scan() {
		newPassphrase2 = scanner.Bytes()
	} else if err := scanner.Err(); err != nil {
		return log.Error(err)
	}
	// compare new passphrases
	if !bytes.Equal(newPassphrase, newPassphrase2) {
		return log.Error("new passphrases differ")
	}
	// rekey keyDB
	log.Infof("rekey keyDB '%s'", keydbname)
	if err := keydb.Rekey(keydbname, oldPassphrase, newPassphrase, iterations); err != nil {
		return err
	}
	return nil
}

func (ce *CryptEngine) dbStatus(w io.Writer) error {
	autoVacuum, freelistCount, err := ce.keyDB.Status()
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "keydb:\n")
	fmt.Fprintf(w, "auto_vacuum=%s\n", autoVacuum)
	fmt.Fprintf(w, "freelist_count=%d\n", freelistCount)
	return nil
}

func (ce *CryptEngine) dbVacuum(autoVacuumMode string) error {
	return ce.keyDB.Vacuum(autoVacuumMode)
}

func (ce *CryptEngine) dbIncremental(pagesToRemove int64) error {
	return ce.keyDB.Incremental(pagesToRemove)
}
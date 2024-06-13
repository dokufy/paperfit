package crypter

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"

	// "crypto/rand"
	"errors"
	"io"
)

var (
	ErrIVwrite = errors.New("could not write initial value")
	ErrIVread  = errors.New("could not read initial value")
)

func EncryptingReader(key string, r io.Reader) (*cipher.StreamReader, error) {
	iv := getIV()
	// if _, err := io.ReadFull(rand.Reader, iv); err != nil {
	// 	return nil, err
	// }

	block, err := newBlock(key)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewOFB(block, iv)
	return &cipher.StreamReader{S: stream, R: r}, nil
}

func DecryptingReader(key string, r io.Reader) (*cipher.StreamReader, error) {
	iv := getIV()

	block, err := newBlock(key)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewOFB(block, iv)
	return &cipher.StreamReader{S: stream, R: r}, nil
}

func newBlock(key string) (cipher.Block, error) {
	hash := md5.Sum([]byte(key))

	block, err := aes.NewCipher(hash[:])
	if err != nil {
		return nil, err
	}

	return block, nil
}

func getIV() []byte {
	return []byte("Wj2uQPvXwG95SzeR")
}

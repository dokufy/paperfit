package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"io"
)

func EncryptingReader(key string, r io.Reader) (*cipher.StreamReader, error) {
	iv := getIV()

	block, err := newBlock(key)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewCTR(block, iv)
	return &cipher.StreamReader{S: stream, R: r}, nil
}

func DecryptingReader(key string, r io.Reader) (*cipher.StreamReader, error) {
	iv := getIV()

	block, err := newBlock(key)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewCTR(block, iv)
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

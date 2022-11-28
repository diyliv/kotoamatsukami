package rsa

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"hash"
	"io"
	"log"
)

func GenerateKeys() (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		log.Printf("error while generating keys: %v\n", err)
		return nil, err
	}
	return privateKey, nil
}

func EncryptOAEP(hash hash.Hash, random io.Reader, pubKey *rsa.PublicKey, msg []byte) ([]byte, error) {
	msgLen := len(msg)
	step := pubKey.Size() - 2*hash.Size() - 2
	var encBytes []byte

	for start := 0; start < msgLen; start += step {
		finish := start + step
		if finish > msgLen {
			finish = msgLen
		}
		encBlockBytes, err := rsa.EncryptOAEP(hash, random, pubKey, msg[start:finish], nil)
		if err != nil {
			return nil, err
		}
		encBytes = append(encBytes, encBlockBytes...)
	}
	return encBytes, nil
}

func DecryptOAEP(hash hash.Hash, random io.Reader, privKey *rsa.PrivateKey, msg []byte) ([]byte, error) {
	msgLen := len(msg)
	step := privKey.PublicKey.Size()
	var decryptedBytes []byte

	for start := 0; start < msgLen; start += step {
		finish := start + step
		if finish > msgLen {
			finish = msgLen
		}

		decryptedBlockBytes, err := rsa.DecryptOAEP(hash, random, privKey, msg[start:finish], nil)
		if err != nil {
			return nil, err
		}

		decryptedBytes = append(decryptedBytes, decryptedBlockBytes...)
	}

	return decryptedBytes, nil
}

func Encrypt(msg []byte, pubKey rsa.PublicKey) string {

	rng := rand.Reader
	cipherText, err := rsa.EncryptOAEP(sha512.New(), rng, &pubKey, []byte(msg), nil)
	if err != nil {
		log.Printf("error while encrypting: %v\n", err)
	}

	return base64.StdEncoding.EncodeToString(cipherText)
}

func Decrypt(encryptedMsg string, privKey *rsa.PrivateKey) string {
	cipherText, err := base64.StdEncoding.DecodeString(encryptedMsg)
	if err != nil {
		log.Printf("error while decoding encrypted message: %v\n", err)
	}

	rng := rand.Reader

	plainText, err := rsa.DecryptOAEP(sha256.New(), rng, privKey, cipherText, nil)
	if err != nil {
		log.Printf("error while decrypting message: %v\n", err)
	}

	return string(plainText)
}

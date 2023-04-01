package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

// Signer is used to create and validate the token.
type Signer struct {
	SecretKey []byte
}

// CheckSign returns the id from the token and a flag indicating if the sign is valid.
func (sg *Signer) CheckSign(s string) (int, bool, error) {
	data, err := hex.DecodeString(s)
	if err != nil {
		return 0, false, err
	}
	id := binary.BigEndian.Uint32(data[:4])
	h := hmac.New(sha256.New, sg.SecretKey)
	h.Write(data[:4])
	sign := h.Sum(nil)
	return int(id), hmac.Equal(sign, data[4:]), nil
}

// CreateSign creates the token from id and its sign generated as HMAC hash of id with SecretKey.
func (sg *Signer) CreateSign(id int) (string, error) {
	data := binary.BigEndian.AppendUint32(nil, uint32(id))
	h := hmac.New(sha256.New, sg.SecretKey)
	h.Write(data)
	sign := h.Sum(nil)
	data = append(data, sign...)
	return hex.EncodeToString(data), nil
}

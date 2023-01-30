package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

var secretKey = []byte("secret key")

func CheckSign(s string) (int, bool, error) {
	data, err := hex.DecodeString(s)
	if err != nil {
		return 0, false, err
	}
	id := binary.BigEndian.Uint32(data[:4])
	h := hmac.New(sha256.New, secretKey)
	h.Write(data[:4])
	sign := h.Sum(nil)
	return int(id), hmac.Equal(sign, data[4:]), nil
}

func CreateSign(id int) (string, error) {
	data := binary.BigEndian.AppendUint32(nil, uint32(id))
	h := hmac.New(sha256.New, secretKey)
	h.Write(data)
	sign := h.Sum(nil)
	data = append(data, sign...)
	return hex.EncodeToString(data), nil
}

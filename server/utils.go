package main

import (
	cryptoRand "crypto/rand"
	"log"
	"math/big"
	"server/since"
)

var colors = [...]string{"#CC241D", "#98971A", "#D79921", "#458588", "#B16286", "#689D6A", "#D65D0E", "#FB4934", "#B8BB26", "#FABD2F", "#83A598", "#D3869B", "#8EC07C", "#FE8019"}

func (s Status) ConvertTime() string {
	return since.Since(s.Timestamp)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	result := make([]byte, length)

	for i := range result {
		n, err := cryptoRand.Int(cryptoRand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			log.Fatal("random string failed???")
			return ""
		}
		result[i] = charset[n.Int64()]
	}

	return string(result)
}

package utils

import (
	"encoding/base64"
	"os"

	"crypto/rand"
)

func FileExists(location string) bool {
	_, err := os.Stat(location)
	return err == nil
}

func GenSecret() string {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "OGv2P1_3nVs_j9"
	}

	secret := base64.URLEncoding.EncodeToString(randomBytes)[:15]

	return string(secret)
}

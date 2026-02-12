package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const tokenExpiry = 30 * 24 * time.Hour // 30 days

func generateToken(password string) (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	nonceHex := hex.EncodeToString(nonce)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	payload := nonceHex + "." + timestamp

	mac := hmac.New(sha256.New, []byte(password))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("%s.%s", payload, sig), nil
}

func validateToken(token, password string) bool {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return false
	}

	nonceHex, timestampStr, sigHex := parts[0], parts[1], parts[2]

	// Verify signature
	payload := nonceHex + "." + timestampStr
	mac := hmac.New(sha256.New, []byte(password))
	mac.Write([]byte(payload))
	expectedSig := mac.Sum(nil)

	actualSig, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}

	if subtle.ConstantTimeCompare(expectedSig, actualSig) != 1 {
		return false
	}

	// Check expiry
	ts, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return false
	}
	if time.Since(time.Unix(ts, 0)) > tokenExpiry {
		return false
	}

	return true
}

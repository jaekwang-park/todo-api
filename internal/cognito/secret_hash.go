package cognito

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

// ComputeSecretHash calculates the SECRET_HASH required by Cognito
// when the app client has a client secret configured.
// Formula: Base64(HMAC_SHA256(clientSecret, username + clientID))
func ComputeSecretHash(username, clientID, clientSecret string) string {
	mac := hmac.New(sha256.New, []byte(clientSecret))
	mac.Write([]byte(username + clientID))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

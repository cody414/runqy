package utilities

import "encoding/base64"

func DecodeBase64OrReturnRaw(input []byte) string {
	decoded, err := base64.StdEncoding.DecodeString(string(input))
	if err != nil {
		// Return raw string if base64 decoding fails
		return string(input)
	}
	return string(decoded)
}

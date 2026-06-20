package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

var ErrInvalidToken = errors.New("invalid page token")

type tokenPayload struct {
	Offset int `json:"offset"`
}

func Offset(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	var payload tokenPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if payload.Offset < 0 {
		return 0, ErrInvalidToken
	}
	return payload.Offset, nil
}

func Token(offset int) string {
	if offset <= 0 {
		return ""
	}
	data, _ := json.Marshal(tokenPayload{Offset: offset})
	return base64.RawURLEncoding.EncodeToString(data)
}

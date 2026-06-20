package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/pagination"
)

const autoFederationPageTokenKind = "ard.autoFederation"

type autoFederationPageState struct {
	Initial           bool
	LocalPageToken    string
	UpstreamPageToken map[string]string
	Buffered          []autoFederationBufferedResult
}

type autoFederationBufferedResult struct {
	Result ard.SearchResult `json:"result"`
	Local  bool             `json:"local,omitempty"`
}

type autoFederationPageTokenPayload struct {
	Kind              string                         `json:"kind"`
	Version           int                            `json:"version"`
	LocalPageToken    string                         `json:"localPageToken,omitempty"`
	UpstreamPageToken map[string]string              `json:"upstreamPageToken,omitempty"`
	Buffered          []autoFederationBufferedResult `json:"buffered,omitempty"`
}

func decodeAutoFederationPageToken(token string) (autoFederationPageState, error) {
	if token == "" {
		return autoFederationPageState{Initial: true}, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return decodeAutoFederationLocalFallback(token)
	}
	var payload autoFederationPageTokenPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return decodeAutoFederationLocalFallback(token)
	}
	if payload.Kind == "" {
		return decodeAutoFederationLocalFallback(token)
	}
	if payload.Kind != autoFederationPageTokenKind || payload.Version != 1 {
		return autoFederationPageState{}, fmt.Errorf("%w: unsupported auto federation token", pagination.ErrInvalidToken)
	}
	if payload.LocalPageToken != "" {
		if _, err := pagination.Offset(payload.LocalPageToken); err != nil {
			return autoFederationPageState{}, err
		}
	}
	if payload.UpstreamPageToken == nil {
		payload.UpstreamPageToken = map[string]string{}
	}
	return autoFederationPageState{
		LocalPageToken:    payload.LocalPageToken,
		UpstreamPageToken: payload.UpstreamPageToken,
		Buffered:          payload.Buffered,
	}, nil
}

func decodeAutoFederationLocalFallback(token string) (autoFederationPageState, error) {
	if _, err := pagination.Offset(token); err != nil {
		return autoFederationPageState{}, err
	}
	return autoFederationPageState{LocalPageToken: token}, nil
}

func encodeAutoFederationPageToken(state autoFederationPageState) string {
	if state.LocalPageToken == "" && len(state.UpstreamPageToken) == 0 && len(state.Buffered) == 0 {
		return ""
	}
	payload := autoFederationPageTokenPayload{
		Kind:              autoFederationPageTokenKind,
		Version:           1,
		LocalPageToken:    state.LocalPageToken,
		UpstreamPageToken: state.UpstreamPageToken,
		Buffered:          state.Buffered,
	}
	data, _ := json.Marshal(payload)
	return base64.RawURLEncoding.EncodeToString(data)
}

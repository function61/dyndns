package main

// Authorization for updating DNS recrord for a hostname. Each hostname gets its own token.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/function61/gokit/os/osutil"
)

type updateTokenValidator struct {
	key []byte
}

func newUpdateTokenValidator() (*updateTokenValidator, error) {
	updateTokenValidatorSecretBase64, err := osutil.GetenvRequired("UPDATE_TOKEN_VALIDATOR_SECRET")
	if err != nil {
		return nil, err
	}

	updateTokenValidatorSecret, err := base64.RawURLEncoding.DecodeString(updateTokenValidatorSecretBase64)
	if err != nil {
		return nil, err
	}

	return &updateTokenValidator{
		key: updateTokenValidatorSecret,
	}, nil
}

func (s *updateTokenValidator) ValidateUpdateToken(forHostname string, givenToken string) error {
	givenTokenBytes, err := base64.RawURLEncoding.DecodeString(givenToken)
	if err != nil {
		return err
	}

	if !hmac.Equal(s.tokenBytesFor(forHostname), givenTokenBytes) {
		return errors.New("auth token does not match hostname")
	}

	return nil
}

func (s *updateTokenValidator) TokenFor(hostname string) string {
	return base64.RawURLEncoding.EncodeToString(s.tokenBytesFor(hostname))
}

func (s *updateTokenValidator) tokenBytesFor(hostname string) []byte {
	mac := hmac.New(sha256.New, s.key)
	if _, err := mac.Write([]byte(hostname)); err != nil {
		panic(err) // shouldn't happen
	}

	return mac.Sum(nil)
}

func getBearerToken(r *http.Request) string {
	authorizationHeader := r.Header.Get("Authorization")

	if strings.HasPrefix(authorizationHeader, "Bearer ") {
		return authorizationHeader[len("Bearer "):]
	} else {
		return ""
	}
}

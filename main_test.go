package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthenticate(t *testing.T) {
	router := initRouter()

	sendAuthRequest := func(data map[string]string, code int, body map[string]string) {
		jsonData, _ := json.Marshal(data)
		bytesData := bytes.NewBuffer(jsonData)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth", bytesData)

		router.ServeHTTP(w, req)

		bodyJson, _ := json.MarshalIndent(body, "", "    ")

		assert.Equal(t, code, w.Code)
		assert.Equal(t, string(bodyJson), w.Body.String())
	}

	noToken := map[string]string{
		"tkn": "",
	}

	noTokenResponse := map[string]string{
		"error": "No token provided",
	}

	sendAuthRequest(noToken, http.StatusBadRequest, noTokenResponse)

	emptyToken := map[string]string{
		"token": "",
	}

	emptyTokenResponse := map[string]string{
		"error": "No token provided",
	}

	sendAuthRequest(emptyToken, http.StatusBadRequest, emptyTokenResponse)

	invalidToken := map[string]string{
		"token": "test",
	}

	invalidTokenResponse := map[string]string{
		"error": "Invalid token",
	}

	sendAuthRequest(invalidToken, http.StatusUnauthorized, invalidTokenResponse)

	validToken := map[string]string{
		"token": os.Getenv("GITHUB_TOKEN"),
	}

	validTokenResponse := map[string]string{
		"user": "reandreev",
	}

	sendAuthRequest(validToken, http.StatusOK, validTokenResponse)
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type APIMessage interface {
	ToString() string
}

type ErrorMessage struct {
	Error string `json:"error"`
}

type AuthenticationMessage struct {
	User string `json:"user"`
}

func (e ErrorMessage) ToString() string {
	jsonData, err := json.MarshalIndent(e, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	return string(jsonData)
}

func (g GitHubRepo) ToString() string {
	jsonData, err := json.MarshalIndent(g, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	return string(jsonData)
}

var randomRepoName string = generateRandomRepoName()

func generateRandomRepoName() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	var n int = 10

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}

func TestAuthenticate(t *testing.T) {
	sendTestRequest := func(data map[string]string, code int, body map[string]string) {
		router := initRouter(false)

		rr := httptest.NewRecorder()

		jsonData, err := json.Marshal(data)
		if err != nil {
			t.Error(err)
		}

		bytesData := bytes.NewBuffer(jsonData)
		req, err := http.NewRequest(http.MethodPost, "/auth", bytesData)
		if err != nil {
			t.Error(err)
		}

		router.ServeHTTP(rr, req)

		bodyJson, err := json.MarshalIndent(body, "", "    ")
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, code, rr.Code)
		assert.Equal(t, string(bodyJson), rr.Body.String())
	}

	t.Run("Missing token field", func(t *testing.T) {
		token := map[string]string{
			"tkn": "",
		}

		response := map[string]string{
			"error": "No token provided",
		}

		sendTestRequest(token, http.StatusBadRequest, response)
	})

	t.Run("Empty token field", func(t *testing.T) {
		token := map[string]string{
			"token": "",
		}

		response := map[string]string{
			"error": "No token provided",
		}

		sendTestRequest(token, http.StatusBadRequest, response)
	})

	t.Run("Invalid token", func(t *testing.T) {
		token := map[string]string{
			"token": "test",
		}

		response := map[string]string{
			"error": "Invalid token",
		}

		sendTestRequest(token, http.StatusUnauthorized, response)
	})

	t.Run("Valid token", func(t *testing.T) {
		token := map[string]string{
			"token": os.Getenv("GITHUB_TOKEN"),
		}

		response := map[string]string{
			"user": "reandreev",
		}

		sendTestRequest(token, http.StatusOK, response)
		resetTokenAndUser()
	})
}

func TestGetRepositories(t *testing.T) {
	sendTestRequest := func(code int, body map[string]string) {
		router := initRouter(false)

		rr := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/repos", nil)

		router.ServeHTTP(rr, req)

		assert.Equal(t, code, rr.Code)
		if body != nil {
			bodyJson, _ := json.MarshalIndent(body, "", "    ")
			assert.Equal(t, string(bodyJson), rr.Body.String())
		}
	}

	t.Run("Unauthenticated", func(t *testing.T) {
		body := map[string]string{
			"error": "Not authenticated",
		}
		sendTestRequest(http.StatusUnauthorized, body)
	})

	t.Run("Authenticated", func(t *testing.T) {
		accessToken.Token = os.Getenv("GITHUB_TOKEN")
		authenticatedUser.Login = "reandreev"

		sendTestRequest(http.StatusOK, nil)
		resetTokenAndUser()
	})
}

func TestCreateRepository(t *testing.T) {
	sendTestRequest := func(data map[string]string, code int, body APIMessage) {
		router := initRouter(false)

		rr := httptest.NewRecorder()

		jsonData, err := json.Marshal(data)
		if err != nil {
			t.Error(err)
		}

		bytesData := bytes.NewBuffer(jsonData)
		req, err := http.NewRequest(http.MethodPost, "/repos", bytesData)
		if err != nil {
			t.Error(err)
		}

		router.ServeHTTP(rr, req)

		bodyJson, err := json.MarshalIndent(body, "", "    ")
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, code, rr.Code)
		assert.Equal(t, string(bodyJson), rr.Body.String())
	}

	t.Run("Unauthenticated", func(t *testing.T) {
		repo := map[string]string{
			"name": "testrepo",
		}

		response := ErrorMessage{"Not authenticated"}

		sendTestRequest(repo, http.StatusUnauthorized, response)
	})

	t.Run("Authenticated: Invalid payload", func(t *testing.T) {
		accessToken.Token = os.Getenv("GITHUB_TOKEN")
		authenticatedUser.Login = "reandreev"

		repo := map[string]string{
			"nameee": "testrepo",
		}

		response := ErrorMessage{"Missing 'name'"}

		sendTestRequest(repo, http.StatusBadRequest, response)
		resetTokenAndUser()
	})

	t.Run("Authenticated: Valid payload - New repo", func(t *testing.T) {
		accessToken.Token = os.Getenv("GITHUB_TOKEN")
		authenticatedUser.Login = "reandreev"

		repo := map[string]string{
			"name": randomRepoName,
		}

		response := GitHubRepo{
			randomRepoName,
			"reandreev/" + randomRepoName,
			"https://github.com/reandreev/" + randomRepoName,
			GitHubUser{
				"reandreev",
				"https://github.com/reandreev",
			},
		}

		sendTestRequest(repo, http.StatusCreated, response)
		resetTokenAndUser()
	})

	t.Run("Authenticated: Valid payload - Existing repo", func(t *testing.T) {
		accessToken.Token = os.Getenv("GITHUB_TOKEN")
		authenticatedUser.Login = "reandreev"

		repo := map[string]string{
			"name": randomRepoName,
		}

		response := GitHubRepo{}

		sendTestRequest(repo, http.StatusUnprocessableEntity, response)
		resetTokenAndUser()
	})
}

func TestDeleteRepository(t *testing.T) {
	sendTestRequest := func(owner string, repo string, code int, body APIMessage) {
		router := initRouter(false)

		rr := httptest.NewRecorder()

		url := fmt.Sprintf("/repos/%s/%s", owner, repo)
		req, err := http.NewRequest(http.MethodDelete, url, nil)
		if err != nil {
			t.Error(err)
		}

		router.ServeHTTP(rr, req)

		assert.Equal(t, code, rr.Code)
		if body != nil {
			bodyJson, err := json.MarshalIndent(body, "", "    ")
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, string(bodyJson), rr.Body.String())
		}
	}
	t.Run("Unauthenticated", func(t *testing.T) {
		response := ErrorMessage{"Not authenticated"}

		sendTestRequest("reandreev", randomRepoName, http.StatusUnauthorized, response)
	})

	t.Run("Authenicated: Success", func(t *testing.T) {
		accessToken.Token = os.Getenv("GITHUB_TOKEN")
		authenticatedUser.Login = "reandreev"

		sendTestRequest("reandreev", randomRepoName, http.StatusNoContent, nil)
		resetTokenAndUser()
	})

	t.Run("Authenicated: Nonexistent", func(t *testing.T) {
		accessToken.Token = os.Getenv("GITHUB_TOKEN")
		authenticatedUser.Login = "reandreev"

		sendTestRequest("reandreev", randomRepoName, http.StatusNotFound, nil)
		resetTokenAndUser()
	})

	t.Run("Authenicated: Not authorized", func(t *testing.T) {
		accessToken.Token = os.Getenv("GITHUB_TOKEN")
		authenticatedUser.Login = "reandreev"

		sendTestRequest("torvalds", "linux", http.StatusForbidden, nil)
		resetTokenAndUser()
	})
}

func TestGetPullRequests(t *testing.T) {
	sendTestRequest := func(owner string, repo string, n int, code int, body map[string]string) {
		router := initRouter(false)

		rr := httptest.NewRecorder()
		url := fmt.Sprintf("/pulls/%s/%s/%d", owner, repo, n)
		req, _ := http.NewRequest(http.MethodGet, url, nil)

		router.ServeHTTP(rr, req)

		assert.Equal(t, code, rr.Code)
		if body != nil {
			bodyJson, _ := json.MarshalIndent(body, "", "    ")
			assert.Equal(t, string(bodyJson), rr.Body.String())
		}
	}

	t.Run("Unauthenticated", func(t *testing.T) {
		body := map[string]string{
			"error": "Not authenticated",
		}

		sendTestRequest("torvalds", "linux", 5, http.StatusUnauthorized, body)
	})

	t.Run("Authenticated", func(t *testing.T) {
		accessToken.Token = os.Getenv("GITHUB_TOKEN")
		authenticatedUser.Login = "reandreev"

		sendTestRequest("torvalds", "linux", 5, http.StatusOK, nil)
		resetTokenAndUser()
	})
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testAccessToken string = os.Getenv("ACCESS_TOKEN")
var testAuthenticatedUser = "reandreev"

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
	sendTestRequest := func(data map[string]string, code int, response APIMessage) {
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

		assert.Equal(t, code, rr.Code)
		assert.Equal(t, response.String(), rr.Body.String())
	}

	t.Run("Missing token field", func(t *testing.T) {
		data := map[string]string{
			"tkn": "",
		}

		response := ResponseMessage{"failure", "No token provided"}

		sendTestRequest(data, http.StatusBadRequest, response)
	})

	t.Run("Empty token field", func(t *testing.T) {
		data := map[string]string{
			"token": "",
		}

		response := ResponseMessage{"failure", "No token provided"}

		sendTestRequest(data, http.StatusBadRequest, response)
	})

	t.Run("Invalid token", func(t *testing.T) {
		data := map[string]string{
			"token": "test",
		}

		response := ResponseMessage{"failure", "Invalid token"}

		sendTestRequest(data, http.StatusUnauthorized, response)
	})

	t.Run("Valid token", func(t *testing.T) {
		data := map[string]string{
			"token": testAccessToken,
		}

		response := ResponseMessage{"success", "Authenticated as " + testAuthenticatedUser}

		sendTestRequest(data, http.StatusOK, response)
		resetTokenAndUser()
	})
}

func TestGetRepositories(t *testing.T) {
	sendTestRequest := func(code int, response APIMessage) {
		router := initRouter(false)

		rr := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/repos", nil)

		router.ServeHTTP(rr, req)

		assert.Equal(t, code, rr.Code)
		if response != nil {
			assert.Equal(t, response.String(), rr.Body.String())
		}
	}

	t.Run("Unauthenticated", func(t *testing.T) {
		response := ResponseMessage{"failure", "Not authenticated"}

		sendTestRequest(http.StatusUnauthorized, response)
	})

	t.Run("Authenticated", func(t *testing.T) {
		accessToken.Token = testAccessToken
		authenticatedUser.Login = testAuthenticatedUser

		sendTestRequest(http.StatusOK, nil)
		resetTokenAndUser()
	})
}

func TestCreateRepository(t *testing.T) {
	sendTestRequest := func(data map[string]string, code int, response APIMessage) {
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

		assert.Equal(t, code, rr.Code)
		assert.Equal(t, response.String(), rr.Body.String())
	}

	t.Run("Unauthenticated", func(t *testing.T) {
		data := map[string]string{
			"name": "testrepo",
		}

		response := ResponseMessage{"failure", "Not authenticated"}

		sendTestRequest(data, http.StatusUnauthorized, response)
	})

	t.Run("Authenticated: Invalid payload", func(t *testing.T) {
		accessToken.Token = testAccessToken
		authenticatedUser.Login = testAuthenticatedUser

		data := map[string]string{
			"nameee": "testrepo",
		}

		response := ResponseMessage{"failure", "Missing 'name'"}

		sendTestRequest(data, http.StatusBadRequest, response)
		resetTokenAndUser()
	})

	t.Run("Authenticated: Valid payload - New repo", func(t *testing.T) {
		accessToken.Token = testAccessToken
		authenticatedUser.Login = testAuthenticatedUser

		data := map[string]string{
			"name": randomRepoName,
		}

		response := GitHubRepo{
			randomRepoName,
			testAuthenticatedUser + "/" + randomRepoName,
			"https://github.com/" + testAuthenticatedUser + "/" + randomRepoName,
			GitHubUser{
				testAuthenticatedUser,
				"https://github.com/" + testAuthenticatedUser,
			},
		}

		sendTestRequest(data, http.StatusCreated, response)
		resetTokenAndUser()
	})

	t.Run("Authenticated: Valid payload - Existing repo", func(t *testing.T) {
		accessToken.Token = testAccessToken
		authenticatedUser.Login = testAuthenticatedUser

		repo := map[string]string{
			"name": randomRepoName,
		}

		response := GitHubRepo{}

		sendTestRequest(repo, http.StatusUnprocessableEntity, response)
		resetTokenAndUser()
	})
}

func TestDeleteRepository(t *testing.T) {
	sendTestRequest := func(owner string, repo string, code int, response APIMessage) {
		router := initRouter(false)

		rr := httptest.NewRecorder()

		url := fmt.Sprintf("/repos/%s/%s", owner, repo)
		req, err := http.NewRequest(http.MethodDelete, url, nil)
		if err != nil {
			t.Error(err)
		}

		router.ServeHTTP(rr, req)

		assert.Equal(t, code, rr.Code)
		assert.Equal(t, response.String(), rr.Body.String())
	}

	t.Run("Unauthenticated", func(t *testing.T) {
		response := ResponseMessage{"failure", "Not authenticated"}

		sendTestRequest(testAuthenticatedUser, randomRepoName, http.StatusUnauthorized, response)
	})

	t.Run("Authenicated: Success", func(t *testing.T) {
		accessToken.Token = testAccessToken
		authenticatedUser.Login = testAuthenticatedUser

		response := ResponseMessage{"success", "Deleted " + randomRepoName}

		sendTestRequest(testAuthenticatedUser, randomRepoName, http.StatusOK, response)
		resetTokenAndUser()
	})

	t.Run("Authenicated: Nonexistent", func(t *testing.T) {
		accessToken.Token = testAccessToken
		authenticatedUser.Login = testAuthenticatedUser

		response := ResponseMessage{"failure", "Repo not found"}

		sendTestRequest(testAuthenticatedUser, randomRepoName, http.StatusNotFound, response)
		resetTokenAndUser()
	})

	t.Run("Authenicated: Not authorized", func(t *testing.T) {
		accessToken.Token = testAccessToken
		authenticatedUser.Login = testAuthenticatedUser

		response := ResponseMessage{"failure", "Not authorized"}

		sendTestRequest("torvalds", "linux", http.StatusForbidden, response)
		resetTokenAndUser()
	})
}

func TestGetPullRequests(t *testing.T) {
	sendTestRequest := func(owner string, repo string, n int, code int, response APIMessage) {
		router := initRouter(false)

		rr := httptest.NewRecorder()
		url := fmt.Sprintf("/pulls/%s/%s/%d", owner, repo, n)
		req, _ := http.NewRequest(http.MethodGet, url, nil)

		router.ServeHTTP(rr, req)

		assert.Equal(t, code, rr.Code)
		if response != nil {
			assert.Equal(t, response.String(), rr.Body.String())
		}
	}

	t.Run("Unauthenticated", func(t *testing.T) {
		response := ResponseMessage{"failure", "Not authenticated"}

		sendTestRequest("torvalds", "linux", 5, http.StatusUnauthorized, response)
	})

	t.Run("Authenticated", func(t *testing.T) {
		accessToken.Token = testAccessToken
		authenticatedUser.Login = testAuthenticatedUser

		sendTestRequest("torvalds", "linux", 5, http.StatusOK, nil)
		resetTokenAndUser()
	})
}

func TestLogout(t *testing.T) {
	sendTestRequest := func(code int, response APIMessage) {
		router := initRouter(false)

		rr := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/logout", nil)

		router.ServeHTTP(rr, req)

		assert.Equal(t, code, rr.Code)
		assert.Equal(t, response.String(), rr.Body.String())
	}

	t.Run("Unauthenticated", func(t *testing.T) {
		response := ResponseMessage{"failure", "Not authenticated"}

		sendTestRequest(http.StatusUnauthorized, response)
	})

	t.Run("Authenticated", func(t *testing.T) {
		accessToken.Token = testAccessToken
		authenticatedUser.Login = testAuthenticatedUser

		response := ResponseMessage{"success", "Logged out"}

		sendTestRequest(http.StatusOK, response)
		resetTokenAndUser()
	})
}

package routes

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"testing"

	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
)

var testAccessToken string = os.Getenv("ACCESS_TOKEN")
var testAuthenticatedUser = "reandreev"
var testJWT string

func generateRandomRepoName() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, 10)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}

func init() {
	jwToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"token": testAccessToken,
	})

	jwtString, _ := jwToken.SignedString(JWT_SECRET)
	testJWT = jwtString
}

func sendTestRequest(t *testing.T, query string, code int, response APIMessage, jwtCookie bool) {
	router := InitRouter(false)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, query, nil)

	if jwtCookie {
		req.AddCookie(&http.Cookie{
			Name:     "jwt",
			Value:    url.QueryEscape(testJWT),
			MaxAge:   0,
			Path:     "/",
			Domain:   "",
			SameSite: 0,
			Secure:   false,
			HttpOnly: true,
		})
	}

	router.ServeHTTP(rr, req)

	assert.Equal(t, code, rr.Code)
	if response != nil {
		assert.Equal(t, response.String(), rr.Body.String())
	}
}

func TestAuthenticate(t *testing.T) {
	t.Parallel()

	t.Run("Missing token parameter I", func(t *testing.T) {
		t.Parallel()
		response := ResponseMessage{http.StatusBadRequest, "No token provided"}

		sendTestRequest(t, "/auth", response.Status, response, false)
	})

	t.Run("Missing token parameter II", func(t *testing.T) {
		t.Parallel()
		response := ResponseMessage{http.StatusBadRequest, "No token provided"}

		sendTestRequest(t, "/auth?tkn=test", response.Status, response, false)
	})

	t.Run("Missing token parameter III", func(t *testing.T) {
		t.Parallel()
		response := ResponseMessage{http.StatusBadRequest, "No token provided"}

		sendTestRequest(t, "/auth?token", response.Status, response, false)
	})

	t.Run("Missing token parameter IV", func(t *testing.T) {
		t.Parallel()
		response := ResponseMessage{http.StatusBadRequest, "No token provided"}

		sendTestRequest(t, "/auth?token=", response.Status, response, false)
	})

	t.Run("Invalid token", func(t *testing.T) {
		t.Parallel()
		response := ResponseMessage{http.StatusUnauthorized, "Invalid token"}

		sendTestRequest(t, "/auth?token=test", response.Status, response, false)
	})

	t.Run("Valid token", func(t *testing.T) {
		t.Parallel()
		response := ResponseMessage{http.StatusOK, "Authenticated as " + testAuthenticatedUser}

		sendTestRequest(t, "/auth?token="+testAccessToken, response.Status, response, false)
	})
}

func TestGetRepositories(t *testing.T) {
	t.Parallel()

	t.Run("Unauthenticated I", func(t *testing.T) {
		t.Parallel()

		response := ResponseMessage{http.StatusUnauthorized, "Not authenticated"}

		sendTestRequest(t, "/repos", response.Status, response, false)
	})

	t.Run("Unauthenticated II", func(t *testing.T) {
		t.Parallel()
		response := ResponseMessage{http.StatusUnauthorized, "Not authenticated"}

		sendTestRequest(t, "/repos/torvalds", response.Status, response, false)
	})

	t.Run("Authenticated: Own repos", func(t *testing.T) {
		t.Parallel()

		sendTestRequest(t, "/repos", http.StatusOK, nil, true)
	})

	t.Run("Authenticated: Someone's repos", func(t *testing.T) {
		t.Parallel()

		sendTestRequest(t, "/repos/torvalds", http.StatusOK, nil, true)
	})

	t.Run("Authenticated: Nonexistent user's repos", func(t *testing.T) {
		t.Parallel()

		sendTestRequest(t, "/repos/abdnsabduihgdç", http.StatusNotFound, nil, true)
	})
}

func TestCreateRepository(t *testing.T) {
	t.Parallel()

	t.Run("Unauthenticated", func(t *testing.T) {
		t.Parallel()

		response := ResponseMessage{http.StatusUnauthorized, "Not authenticated"}

		sendTestRequest(t, "/repos/create?name=test", response.Status, response, false)
	})

	t.Run("Authenticated - Missing name parameter I", func(t *testing.T) {
		t.Parallel()

		response := ResponseMessage{http.StatusBadRequest, "Missing name parameter"}

		sendTestRequest(t, "/repos/create", response.Status, response, true)
	})

	t.Run("Authenticated - Missing name parameter II", func(t *testing.T) {
		t.Parallel()
		response := ResponseMessage{http.StatusBadRequest, "Missing name parameter"}

		sendTestRequest(t, "/repos/create?nam=", response.Status, response, true)
	})

	t.Run("Authenticated - Missing name parameter III", func(t *testing.T) {
		t.Parallel()
		response := ResponseMessage{http.StatusBadRequest, "Missing name parameter"}

		sendTestRequest(t, "/repos/create?name=", response.Status, response, true)
	})

	t.Run("Authenticated - New repo", func(t *testing.T) {
		t.Parallel()

		repoName := generateRandomRepoName()

		t.Cleanup(func() {
			cmd := exec.Command(
				"curl", "-L",
				"-X", "DELETE",
				"-H", "Accept: application/vnd.github+json",
				"-H", "Authorization: Bearer "+testAccessToken,
				"-H", "X-GitHub-Api-Version: 2022-11-28",
				"https://api.github.com/repos/"+testAuthenticatedUser+"/"+repoName,
			)

			if err := cmd.Run(); err != nil {
				t.Fatal(err)
			}
		})

		sendTestRequest(t, "/repos/create?name="+repoName, http.StatusOK, nil, true)
	})

	t.Run("Authenticated - Already existing repo", func(t *testing.T) {
		t.Parallel()

		repoName := generateRandomRepoName()

		t.Cleanup(func() {
			cmd := exec.Command(
				"curl", "-L",
				"-X", "DELETE",
				"-H", "Accept: application/vnd.github+json",
				"-H", "Authorization: Bearer "+testAccessToken,
				"-H", "X-GitHub-Api-Version: 2022-11-28",
				"https://api.github.com/repos/"+testAuthenticatedUser+"/"+repoName,
			)

			if err := cmd.Run(); err != nil {
				t.Fatal(err)
			}
		})

		cmd := exec.Command(
			"curl", "-L",
			"-X", "POST",
			"-H", "Accept: application/vnd.github+json",
			"-H", "Authorization: Bearer "+testAccessToken,
			"-H", "X-GitHub-Api-Version: 2022-11-28",
			"https://api.github.com/user/repos",
			"-d", "{\"name\": \""+repoName+"\"}",
		)

		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		response := ResponseMessage{http.StatusUnprocessableEntity, "Repo already exists"}

		sendTestRequest(t, "/repos/create?name="+repoName, response.Status, response, true)
	})
}

func TestDeleteRepository(t *testing.T) {
	t.Parallel()

	t.Run("Unauthenticated", func(t *testing.T) {
		t.Parallel()

		query := "/repos/delete/" + testAuthenticatedUser + "/" + generateRandomRepoName()
		response := ResponseMessage{http.StatusUnauthorized, "Not authenticated"}

		sendTestRequest(t, query, response.Status, response, false)
	})

	t.Run("Authenticated - Success", func(t *testing.T) {
		t.Parallel()

		repoName := generateRandomRepoName()

		cmd := exec.Command(
			"curl", "-L",
			"-X", "POST",
			"-H", "Accept: application/vnd.github+json",
			"-H", "Authorization: Bearer "+testAccessToken,
			"-H", "X-GitHub-Api-Version: 2022-11-28",
			"https://api.github.com/user/repos",
			"-d", "{\"name\": \""+repoName+"\"}",
		)

		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		query := "/repos/delete/" + testAuthenticatedUser + "/" + repoName
		response := ResponseMessage{http.StatusOK, "Deleted " + repoName}

		sendTestRequest(t, query, response.Status, response, true)
	})

	t.Run("Authenticated - Nonexistent", func(t *testing.T) {
		t.Parallel()

		repoName := generateRandomRepoName()

		query := "/repos/delete/" + testAuthenticatedUser + "/" + repoName
		response := ResponseMessage{http.StatusNotFound, "Repo not found"}

		sendTestRequest(t, query, response.Status, response, true)
	})

	t.Run("Authenticated - Not authorized", func(t *testing.T) {
		t.Parallel()

		query := "/repos/delete/torvalds/linux"
		response := ResponseMessage{http.StatusForbidden, "Not authorized"}

		sendTestRequest(t, query, response.Status, response, true)
	})
}

func TestGetPullRequests(t *testing.T) {
	t.Parallel()

	t.Run("Unauthenticated", func(t *testing.T) {
		t.Parallel()

		query := "/pulls/torvalds/linux/5"
		response := ResponseMessage{http.StatusUnauthorized, "Not authenticated"}

		sendTestRequest(t, query, response.Status, response, false)
	})

	t.Run("Authenticated - Success", func(t *testing.T) {
		t.Parallel()

		query := "/pulls/torvalds/linux/5"

		sendTestRequest(t, query, http.StatusOK, nil, true)
	})

	t.Run("Authenticated - Success", func(t *testing.T) {
		t.Parallel()

		query := "/pulls/torvalds/linuxx/5"
		response := ResponseMessage{http.StatusNotFound, "Repo not found"}

		sendTestRequest(t, query, response.Status, response, true)
	})
}

func TestLogout(t *testing.T) {
	t.Parallel()

	t.Run("Unauthenticated", func(t *testing.T) {
		t.Parallel()

		response := ResponseMessage{http.StatusUnauthorized, "Not authenticated"}

		sendTestRequest(t, "/logout", response.Status, response, false)
	})

	t.Run("Authenticated", func(t *testing.T) {
		t.Parallel()

		response := ResponseMessage{http.StatusOK, "Logged out"}

		sendTestRequest(t, "/logout", response.Status, response, true)
	})
}

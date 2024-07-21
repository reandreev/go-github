package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

type APIMessage interface {
	String() string
}

type ResponseMessage struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func String(a any) string {
	jsonData, err := json.MarshalIndent(a, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	return string(jsonData)
}

func (e ResponseMessage) String() string {
	return String(e)
}

func (g GitHubRepo) String() string {
	return String(g)
}

type GitHubUser struct {
	Login   string `json:"login"`
	HtmlUrl string `json:"html_url"`
}

type GitHubRepo struct {
	Name     string     `json:"name"`
	FullName string     `json:"full_name"`
	HtmlUrl  string     `json:"html_url"`
	Owner    GitHubUser `json:"owner"`
}

type GitHubPullRequest struct {
	Number int        `json:"number"`
	Title  string     `json:"title"`
	User   GitHubUser `json:"user"`
}

const INDENT_JSON bool = true

var JWT_SECRET []byte = []byte("secret_key")

func InitRouter(logging bool) *gin.Engine {
	var router *gin.Engine

	if logging {
		router = gin.Default()
	} else {
		gin.SetMode(gin.ReleaseMode)
		router = gin.New()
	}

	router.POST("/auth", authenticate)

	authenticated := router.Group("/", func(c *gin.Context) {
		if cookie, err := c.Cookie("jwt"); err != nil || cookie == "" {
			sendJSON(c, http.StatusUnauthorized, "Not authenticated", INDENT_JSON)
			c.Abort()
		} else {
			token, err := jwt.Parse(cookie, func(token *jwt.Token) (interface{}, error) {
				return JWT_SECRET, nil
			})

			if err != nil {
				sendJSON(c, http.StatusUnauthorized, "Bad JWT 1", INDENT_JSON)
				c.Abort()
			} else if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				c.Set("token", claims["token"])
				c.Set("user", claims["user"])
				c.Next()
			} else {
				sendJSON(c, http.StatusUnauthorized, "Bad JWT 2", INDENT_JSON)
				c.Abort()
			}
		}
	})

	authenticated.GET("/auth", user)
	authenticated.DELETE("/auth", logout)

	authenticated.GET("/repos", getRepositories)
	authenticated.GET("/repos/:user", getRepositories)
	authenticated.POST("/repos", createRepository)

	authenticated.DELETE("/repos/:owner/:repo", deleteRepository)

	authenticated.GET("/pulls/:owner/:repo/:n", getPullRequests)

	return router
}

func authenticate(c *gin.Context) {
	token := c.Query("token")

	if token == "" {
		sendJSON(c, http.StatusBadRequest, "No token provided", INDENT_JSON)
		return
	}

	resp, err := sendRequest(token, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		sendJSON(c, http.StatusInternalServerError, "Unexpected error", INDENT_JSON)
		return
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNotModified:
		authenticatedUser := GitHubUser{}

		err := json.NewDecoder(resp.Body).Decode(&authenticatedUser)
		if err != nil {
			sendJSON(c, http.StatusInternalServerError, "Unable to parse user info", INDENT_JSON)
			return
		}

		jwToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"token": token,
		})

		jwtString, err := jwToken.SignedString(JWT_SECRET)
		if err != nil {
			sendJSON(c, http.StatusInternalServerError, "Unable to generate JWT", INDENT_JSON)
			return
		}

		c.SetCookie("jwt", jwtString, 300, "/", "", false, true)
		sendJSON(c, http.StatusOK, "Authenticated as "+authenticatedUser.Login, INDENT_JSON)
	case http.StatusForbidden:
		ratelimit(c, resp)
	case http.StatusUnauthorized:
		sendJSON(c, resp.StatusCode, "Invalid token", INDENT_JSON)
	default:
		sendJSON(c, resp.StatusCode, "Unexpected status code", INDENT_JSON)
	}
}

func user(c *gin.Context) {
	token := c.MustGet("token").(string)

	defer delete(c.Keys, "token")

	resp, err := sendRequest(token, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		sendJSON(c, http.StatusInternalServerError, "Unexpected error", INDENT_JSON)
		return
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNotModified:
		authenticatedUser := GitHubUser{}

		err := json.NewDecoder(resp.Body).Decode(&authenticatedUser)
		if err != nil {
			sendJSON(c, http.StatusInternalServerError, "Unable to parse user info", INDENT_JSON)
			return
		}

		sendJSON(c, http.StatusOK, "", INDENT_JSON, authenticatedUser)
	case http.StatusForbidden:
		ratelimit(c, resp)
	case http.StatusUnauthorized:
		sendJSON(c, resp.StatusCode, "Invalid token", INDENT_JSON)
	default:
		sendJSON(c, resp.StatusCode, "Unexpected status code", INDENT_JSON)
	}
}

func logout(c *gin.Context) {
	c.SetCookie("jwt", "", 0, "/", "", false, true)
	sendJSON(c, http.StatusOK, "Logged out", INDENT_JSON)
}

func getRepositories(c *gin.Context) {
	var repos []GitHubRepo
	var url string

	if user := c.Param("user"); user != "" {
		url = fmt.Sprintf("https://api.github.com/users/%s/repos", user)
	} else {
		url = "https://api.github.com/user/repos"
	}

	token := c.MustGet("token").(string)

	defer delete(c.Keys, "token")

	resp, err := sendRequest(token, http.MethodGet, url, nil)
	if err != nil {
		sendJSON(c, http.StatusInternalServerError, "Unexpected error", INDENT_JSON)
		return
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNotModified:
		err := json.NewDecoder(resp.Body).Decode(&repos)
		if err != nil {
			sendJSON(c, http.StatusInternalServerError, "Unable to parse repos info", INDENT_JSON)
			return
		}

		sendJSON(c, http.StatusOK, "", INDENT_JSON, repos)
	case http.StatusUnauthorized:
		sendJSON(c, resp.StatusCode, "Not authenticated", INDENT_JSON)
	case http.StatusForbidden:
		ratelimit(c, resp)
	case http.StatusNotFound:
		sendJSON(c, resp.StatusCode, "User not found", INDENT_JSON)
	case http.StatusUnprocessableEntity:
		sendJSON(c, resp.StatusCode, "Malformed JSON", INDENT_JSON)
	default:
		sendJSON(c, resp.StatusCode, "Unexpected status code", INDENT_JSON)
	}
}

func createRepository(c *gin.Context) {
	name := c.Query("name")

	if name == "" {
		sendJSON(c, http.StatusBadRequest, "Missing name parameter", INDENT_JSON)
		return
	}

	repoCreationPayload := map[string]any{
		"name": name,
	}

	for k, v := range c.Request.URL.Query() {
		if value, err := strconv.ParseInt(v[0], 10, 64); err == nil {
			repoCreationPayload[k] = value
		} else if value, err := strconv.ParseBool(v[0]); err == nil {
			repoCreationPayload[k] = value
		} else {
			repoCreationPayload[k] = v[0]
		}
	}

	jsonRepoCreationPayload, err := json.Marshal(repoCreationPayload)
	if err != nil {
		sendJSON(c, http.StatusInternalServerError, "Unexpected error json", INDENT_JSON)
		return
	}

	token := c.MustGet("token").(string)

	defer delete(c.Keys, "token")

	resp, err := sendRequest(token, http.MethodPost, "https://api.github.com/user/repos", bytes.NewBuffer(jsonRepoCreationPayload))
	if err != nil {
		sendJSON(c, http.StatusInternalServerError, "Unexpected error", INDENT_JSON)
		return
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated:
		var repo map[string]any
		err := json.NewDecoder(resp.Body).Decode(&repo)
		if err != nil {
			sendJSON(c, http.StatusInternalServerError, "Unable to parse repo info", INDENT_JSON)
			return
		}

		sendJSON(c, http.StatusOK, "", INDENT_JSON, repo)
	case http.StatusNotModified:
		sendJSON(c, resp.StatusCode, "Not modified", INDENT_JSON)
	case http.StatusBadRequest:
		sendJSON(c, resp.StatusCode, "Bad request", INDENT_JSON)
	case http.StatusUnauthorized:
		sendJSON(c, resp.StatusCode, "Not authenticated", INDENT_JSON)
	case http.StatusForbidden:
		ratelimit(c, resp)
	case http.StatusNotFound:
		sendJSON(c, resp.StatusCode, "Resource not found", INDENT_JSON)
	case http.StatusUnprocessableEntity:
		sendJSON(c, resp.StatusCode, "Repo already exists", INDENT_JSON)
	default:
		sendJSON(c, resp.StatusCode, "Unexpected status code", INDENT_JSON)
	}
}

func deleteRepository(c *gin.Context) {
	token := c.MustGet("token").(string)

	owner := c.Param("owner")
	repo := c.Param("repo")
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)

	defer delete(c.Keys, "token")

	resp, err := sendRequest(token, http.MethodDelete, url, nil)
	if err != nil {
		sendJSON(c, http.StatusInternalServerError, "Unexpected error", INDENT_JSON)
		return
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		sendJSON(c, http.StatusOK, "Deleted "+repo, INDENT_JSON)
	case http.StatusTemporaryRedirect:
		sendJSON(c, resp.StatusCode, "Temporary redirect", INDENT_JSON)
	case http.StatusForbidden:
		remaining, err := strconv.ParseInt(resp.Header.Get("x-ratelimit-remaining"), 10, 64)
		if err != nil || remaining > 0 {
			sendJSON(c, resp.StatusCode, "Not authorized", INDENT_JSON)
		} else {
			ratelimit(c, resp)
		}
	case http.StatusNotFound:
		sendJSON(c, resp.StatusCode, "Repo not found", INDENT_JSON)
	default:
		sendJSON(c, resp.StatusCode, "Unexpected status code", INDENT_JSON)
	}
}

func getPullRequests(c *gin.Context) {
	var pullRequests []GitHubPullRequest

	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/pulls?per_page=%s",
		c.Param("owner"),
		c.Param("repo"),
		c.Param("n"),
	)

	token := c.MustGet("token").(string)

	defer delete(c.Keys, "token")

	resp, err := sendRequest(token, http.MethodGet, url, nil)
	if err != nil {
		sendJSON(c, http.StatusInternalServerError, "Unexpected error", INDENT_JSON)
		return
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		err := json.NewDecoder(resp.Body).Decode(&pullRequests)
		if err != nil {
			sendJSON(c, http.StatusInternalServerError, "Unable to parse PR info", INDENT_JSON)
			return
		}

		sendJSON(c, http.StatusOK, "", INDENT_JSON, pullRequests)
	case http.StatusNotModified:
		sendJSON(c, resp.StatusCode, "Not modified", INDENT_JSON)
	case http.StatusNotFound:
		sendJSON(c, resp.StatusCode, "Repo not found", INDENT_JSON)
	case http.StatusUnprocessableEntity:
		sendJSON(c, resp.StatusCode, "Endpoint spam", INDENT_JSON)
	default:
		sendJSON(c, resp.StatusCode, "Unexpected status code", INDENT_JSON)
	}
}

package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIMessage interface {
	String() string
}

type ResponseMessage struct {
	Result  string `json:"result"`
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

type GitHubToken struct {
	Token string `json:"token" binding:"required"`
}

type GitHubRepoCreation struct {
	Name string `json:"name" binding:"required"`
}

var accessToken GitHubToken
var authenticatedUser GitHubUser

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
		if authenticatedUser.Login == "" {
			c.IndentedJSON(http.StatusUnauthorized, ResponseMessage{"failure", "Not authenticated"})
			c.Abort()
		} else {
			c.Next()
		}
	})

	authenticated.GET("/repos", getRepositories)
	authenticated.GET("/repos/:user", getRepositories)
	authenticated.POST("/repos", createRepository)
	authenticated.DELETE("/repos/:owner/:repo", deleteRepository)
	authenticated.GET("/pulls/:owner/:repo/:n", getPullRequests)
	authenticated.GET("/logout", logout)

	return router
}

func authenticate(c *gin.Context) {
	if err := c.ShouldBindJSON(&accessToken); err != nil {
		c.IndentedJSON(http.StatusBadRequest, ResponseMessage{"failure", "No token provided"})
		return
	}

	resp, err := sendRequest(http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, ResponseMessage{"failure", "Unexpected error"})
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		err := json.NewDecoder(resp.Body).Decode(&authenticatedUser)
		if err != nil {
			authenticatedUser = GitHubUser{}
			c.IndentedJSON(http.StatusInternalServerError, ResponseMessage{"failure", "Unexpected error"})
			return
		}

		c.IndentedJSON(resp.StatusCode, ResponseMessage{"success", "Authenticated as " + authenticatedUser.Login})
	} else {
		accessToken = GitHubToken{}
		c.IndentedJSON(http.StatusUnauthorized, ResponseMessage{"failure", "Invalid token"})
	}
}

func resetTokenAndUser() {
	accessToken = GitHubToken{}
	authenticatedUser = GitHubUser{}
}

func logout(c *gin.Context) {
	resetTokenAndUser()
	c.IndentedJSON(http.StatusOK, ResponseMessage{"success", "Logged out"})
}

func getRepositories(c *gin.Context) {
	var repos []GitHubRepo
	var url string

	if user := c.Param("user"); user != "" {
		url = fmt.Sprintf("https://api.github.com/users/%s/repos", user)
	} else {
		url = "https://api.github.com/user/repos"
	}

	resp, err := sendRequest(http.MethodGet, url, nil)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, ResponseMessage{"failure", "Unexpected error"})
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		err := json.NewDecoder(resp.Body).Decode(&repos)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, ResponseMessage{"failure", "Unexpected error"})
			return
		}
	}

	c.IndentedJSON(resp.StatusCode, repos)
}

func createRepository(c *gin.Context) {
	var data GitHubRepoCreation
	var repo GitHubRepo

	if err := c.BindJSON(&data); err != nil {
		c.IndentedJSON(http.StatusBadRequest, ResponseMessage{"failure", "Missing 'name'"})
		return
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, ResponseMessage{"failure", "Unexpected error"})
		return
	}

	bytesData := bytes.NewBuffer(jsonData)
	resp, err := sendRequest(http.MethodPost, "https://api.github.com/user/repos", bytesData)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, ResponseMessage{"failure", "Unexpected error"})
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		err := json.NewDecoder(resp.Body).Decode(&repo)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, ResponseMessage{"failure", "Unexpected error"})
			return
		}
	}

	c.IndentedJSON(resp.StatusCode, repo)
}

func deleteRepository(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)

	resp, err := sendRequest(http.MethodDelete, url, nil)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, ResponseMessage{"failure", "Unexpected error"})
		return
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		c.IndentedJSON(http.StatusOK, ResponseMessage{"success", "Deleted " + repo})
	case http.StatusNotFound:
		c.IndentedJSON(resp.StatusCode, ResponseMessage{"failure", "Repo not found"})
	case http.StatusForbidden:
		c.IndentedJSON(resp.StatusCode, ResponseMessage{"failure", "Not authorized"})
	default:
		c.IndentedJSON(resp.StatusCode, ResponseMessage{"redirected", ""})
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

	resp, err := sendRequest(http.MethodGet, url, nil)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, ResponseMessage{"failure", "Unexpected error"})
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		err := json.NewDecoder(resp.Body).Decode(&pullRequests)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, ResponseMessage{"failure", "Unexpected error"})
			return
		}
	}

	c.IndentedJSON(resp.StatusCode, pullRequests)
}

func sendRequest(method string, url string, body io.Reader) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, body)

	if err != nil {
		return nil, err
	}

	req.Header = http.Header{
		"Accept":        {"application/vnd.github+json"},
		"Authorization": {"Bearer " + accessToken.Token},
	}

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

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

func main() {
	initRouter(true).Run(":8080")
}

func initRouter(logging bool) *gin.Engine {
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
			c.IndentedJSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
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
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "No token provided"})
		return
	}

	resp := sendRequest(http.MethodGet, "https://api.github.com/user", nil)

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		json.NewDecoder(resp.Body).Decode(&authenticatedUser)
		c.IndentedJSON(resp.StatusCode, gin.H{"user": authenticatedUser.Login})
	} else {
		accessToken = GitHubToken{}
		c.IndentedJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
	}
}

func resetTokenAndUser() {
	accessToken = GitHubToken{}
	authenticatedUser = GitHubUser{}
}

func logout(c *gin.Context) {
	resetTokenAndUser()
	c.String(http.StatusOK, "Logged out")
}

func getRepositories(c *gin.Context) {
	var repos []GitHubRepo
	var url string

	if user := c.Param("user"); user != "" {
		url = fmt.Sprintf("https://api.github.com/users/%s/repos", user)
	} else {
		url = "https://api.github.com/user/repos"
	}

	resp := sendRequest(http.MethodGet, url, nil)

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		json.NewDecoder(resp.Body).Decode(&repos)
	}

	c.IndentedJSON(resp.StatusCode, repos)
}

func createRepository(c *gin.Context) {
	var data GitHubRepoCreation
	var repo GitHubRepo

	if err := c.BindJSON(&data); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "Missing 'name'"})
		return
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}

	bytesData := bytes.NewBuffer(jsonData)
	resp := sendRequest(http.MethodPost, "https://api.github.com/user/repos", bytesData)

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
		json.NewDecoder(resp.Body).Decode(&repo)
	}

	c.IndentedJSON(resp.StatusCode, repo)
}

func deleteRepository(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)

	resp := sendRequest(http.MethodDelete, url, nil)

	c.String(resp.StatusCode, fmt.Sprintln(resp.StatusCode))
}

func getPullRequests(c *gin.Context) {
	var pullRequests []GitHubPullRequest

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls?per_page=%s", c.Param("owner"), c.Param("repo"), c.Param("n"))

	resp := sendRequest(http.MethodGet, url, nil)

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		json.NewDecoder(resp.Body).Decode(&pullRequests)
	}

	c.IndentedJSON(resp.StatusCode, pullRequests)
}

func sendRequest(method string, url string, body io.Reader) *http.Response {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, body)

	if err != nil {
		log.Fatal(err)
	}

	req.Header = http.Header{
		"Accept":        {"application/vnd.github+json"},
		"Authorization": {"Bearer " + accessToken.Token},
	}

	resp, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}

	return resp
}

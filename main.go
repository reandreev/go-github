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

var accessToken string = "" //os.Getenv("GITHUB_TOKEN")

func main() {
	router := gin.Default()

	router.POST("/auth", authenticate)

	router.GET("/repos", getRepositories)
	router.GET("/repos/:user", getRepositories)
	router.POST("/repos", createRepository)
	router.DELETE("/repos/:owner/:repo", deleteRepository)
	router.GET("/pulls/:owner/:repo/:n", getPullRequests)

	router.Run(":8080")
}

func authenticate(c *gin.Context) {
	var data map[string]string

	if err := c.BindJSON(&data); err != nil {
		log.Fatal(err)
	}

	accessToken = data["token"]

	resp := sendRequest(http.MethodGet, "https://api.github.com/user", nil)

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		var user GitHubUser
		json.NewDecoder(resp.Body).Decode(&user)
		c.String(resp.StatusCode, fmt.Sprintf("Authenticated as %s", user.Login))
	} else {
		c.String(resp.StatusCode, "Error")
	}
}

func getRepositories(c *gin.Context) {
	if accessToken == "" {
		c.String(http.StatusUnauthorized, "Authenticate with /auth\n")
		return
	}

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
	if accessToken == "" {
		c.String(http.StatusUnauthorized, "Authenticate with /auth\n")
		return
	}

	var data map[string]string
	var repo GitHubRepo

	if err := c.BindJSON(&data); err != nil {
		log.Fatal(err)
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
	if accessToken == "" {
		c.String(http.StatusUnauthorized, "Authenticate with /auth\n")
		return
	}

	owner := c.Param("owner")
	repo := c.Param("repo")
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)

	resp := sendRequest(http.MethodDelete, url, nil)

	c.String(resp.StatusCode, fmt.Sprintln(resp.StatusCode))
}

func getPullRequests(c *gin.Context) {
	if accessToken == "" {
		c.String(http.StatusUnauthorized, "Authenticate with /auth\n")
		return
	}

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
		"Authorization": {"Bearer " + accessToken},
	}

	resp, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}

	return resp
}

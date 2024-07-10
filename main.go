package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type GitHubUser struct {
	Login    string `json:"login"`
	Url      string `json:"url"`
	HtmlUrl  string `json:"html_url"`
	ReposUrl string `json:"repos_url"`
}

type GitHubRepo struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	HtmlUrl  string `json:"html_url"`
}

var accessToken string = os.Getenv("GITHUB_TOKEN")

func main() {
	// repoName := "RandomRepoName123"

	// createRepository(repoName)

	// var user GitHubUser = getUserInfo()
	// deleteRepository(user.Login, repoName)

	repos := getRepositories()

	for _, repo := range repos {
		fmt.Printf("Repo: %+v\n", repo)
	}
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

	if method != "GET" {
		defer resp.Body.Close()
	}

	return resp
}

func getUserInfo() GitHubUser {
	var user GitHubUser
	resp := sendRequest("GET", "https://api.github.com/user", nil)

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		json.NewDecoder(resp.Body).Decode(&user)
	}

	return user
}

func createRepository(name string) {
	data := map[string]interface{}{
		"name": name,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}

	bytesData := bytes.NewBuffer(jsonData)
	resp := sendRequest("POST", "https://api.github.com/user/repos", bytesData)

	if resp.StatusCode == http.StatusCreated {
		fmt.Printf("Repo %v created\n", name)
	} else {
		fmt.Println(resp.StatusCode)
	}
}

func deleteRepository(owner string, name string) {
	resp := sendRequest("DELETE", "https://api.github.com/repos/"+owner+"/"+name, nil)

	if resp.StatusCode == http.StatusNoContent {
		fmt.Printf("Repo %v deleted\n", name)
	} else {
		fmt.Println(resp.StatusCode)
	}
}

func getRepositories() []GitHubRepo {
	var repos []GitHubRepo
	resp := sendRequest("GET", "https://api.github.com/user/repos", nil)

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		json.NewDecoder(resp.Body).Decode(&repos)
	}

	return repos
}

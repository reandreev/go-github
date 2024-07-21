package routes

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func sendJSON(c *gin.Context, code int, message string, indent bool, retrieved ...any) {
	var response any

	if len(retrieved) == 0 {
		response = ResponseMessage{code, message}
	} else if len(retrieved) == 1 {
		response = retrieved[0]
	} else {
		response = retrieved
	}

	if indent {
		c.IndentedJSON(code, response)
	} else {
		c.JSON(code, response)
	}
}

func ratelimit(c *gin.Context, resp *http.Response) {
	resetTime, err := strconv.ParseInt(resp.Header.Get("x-ratelimit-reset"), 10, 64)
	if err != nil {
		sendJSON(c, resp.StatusCode, "Exceeded rate limit. Try again later", INDENT_JSON)
	} else {
		timeLeft := time.Until(time.Unix(resetTime, 0)).Round(time.Second).String()
		sendJSON(c, resp.StatusCode, "Exceeded rate limit. Try again in "+timeLeft, INDENT_JSON)
	}
}

func sendRequest(token string, method string, url string, body io.Reader) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, body)

	if err != nil {
		return nil, err
	}

	req.Header = http.Header{
		"Accept":               {"application/vnd.github+json"},
		"Authorization":        {"Bearer " + token},
		"X-GitHub-Api-Version": {"2022-11-28"},
	}

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

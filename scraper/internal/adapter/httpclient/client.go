package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Client struct {
	BaseClient      *http.Client
	FlareSolverrURL string
}

func New() *Client {
	return &Client{
		BaseClient: &http.Client{
			Timeout: 65 * time.Second, // Long timeout for FlareSolverr
		},
		FlareSolverrURL: os.Getenv("FLARESOLVERR_URL"),
	}
}

type FlareSolverrRequest struct {
	Cmd        string `json:"cmd"`
	URL        string `json:"url"`
	MaxTimeout int    `json:"maxTimeout"`
}

type FlareSolverrResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Solution struct {
		URL      string `json:"url"`
		Status   int    `json:"status"`
		Response string `json:"response"`
	} `json:"solution"`
}

// Get fetches the URL. If useFlareSolverr is true and FlareSolverrURL is configured, it routes through FlareSolverr.
func (c *Client) Get(ctx context.Context, url string, useFlareSolverr bool) (string, error) {
	if useFlareSolverr && c.FlareSolverrURL != "" {
		return c.getFromFlareSolverr(ctx, url)
	}
	return c.getDirect(ctx, url)
}

func (c *Client) getDirect(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	
	// Add default user agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := c.BaseClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *Client) getFromFlareSolverr(ctx context.Context, url string) (string, error) {
	reqBody := FlareSolverrRequest{
		Cmd:        "request.get",
		URL:        url,
		MaxTimeout: 60000,
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// FlareSolverr can take a while to solve challenges
	fsCtx, cancel := context.WithTimeout(ctx, 65*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(fsCtx, "POST", c.FlareSolverrURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.BaseClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var fsResp FlareSolverrResponse
	if err := json.Unmarshal(body, &fsResp); err != nil {
		return "", fmt.Errorf("failed to decode flaresolverr response: %w. Body: %s", err, string(body))
	}

	if fsResp.Status != "ok" {
		return "", fmt.Errorf("flaresolverr error: %s", fsResp.Message)
	}

	if fsResp.Solution.Status < 200 || fsResp.Solution.Status >= 300 {
		return "", fmt.Errorf("unexpected status code from flaresolverr destination: %d", fsResp.Solution.Status)
	}

	return fsResp.Solution.Response, nil
}

// Transport implements http.RoundTripper for dropping into Colly
type Transport struct {
	Client          *Client
	UseFlareSolverr bool
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	html, err := t.Client.Get(req.Context(), req.URL.String(), t.UseFlareSolverr)
	if err != nil {
		return nil, err
	}

	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte(html))),
		Request:    req,
		Header:     make(http.Header),
	}, nil
}

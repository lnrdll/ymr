package source

import (
	"fmt"
	"io"
	"net/http"
)

// FetchHTTP fetches content from a URL, with optional GitHub token authentication.
func FetchHTTP(url string, token string, isSpec bool) ([]byte, error) {
	transformedURL := transformURL(url, isSpec)

	client := &http.Client{}
	req, err := http.NewRequest("GET", transformedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", transformedURL, err)
	}

	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}
	req.Header.Add("Accept", "application/vnd.github.v3.raw")
	req.Header.Add("Cache-control", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote content from %s: %w", transformedURL, err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			if err == nil {
				err = cerr
			}
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad response from server for %s: %s", transformedURL, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

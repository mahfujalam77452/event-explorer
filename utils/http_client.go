package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

type HTTPError struct {
	StatusCode int
	Body string
}

func (e *HTTPError)Error() string {
	return fmt.Sprintf("upstream returned status %d",e.StatusCode)
}

func NewHTTPClient (timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

// A common function for make HTTP request and get the response in go Struct 

func DoJSON(client *http.Client, req *http.Request, out interface{}) error {
	resp, err := client.Do(req)
	if err != nil {
		var uerr *url.Error
		if errors.As(err, &uerr) {
			return fmt.Errorf("request to %s failed: %w", req.URL.Host, uerr.Err)
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Body:string(body),
		}
	}

	if out == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func IsTimeout (err error) bool {
	var ne net.Error

	return errors.As(err,&ne) && ne.Timeout()
}


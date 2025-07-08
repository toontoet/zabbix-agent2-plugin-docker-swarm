package main

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.zabbix.com/sdk/zbxerr"
)

const dockerAPIVersion = "v1.41"

type client struct {
	client http.Client
}

func newClient(socketPath string, timeout int) *client {
	transport := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}

	return &client{
		client: http.Client{
			Transport: transport,
			Timeout:   time.Duration(timeout) * time.Second,
		},
	}
}

func (cli *client) Query(path string, filters map[string][]string) ([]byte, error) {
	u := url.URL{
		Scheme: "http",
		Host:   "localhost", // host is irrelevant for unix sockets
		Path:   dockerAPIVersion + "/" + path,
	}

	if filters != nil {
		filterJSON, err := json.Marshal(filters)
		if err != nil {
			return nil, zbxerr.ErrorCannotMarshalJSON.Wrap(err)
		}
		q := u.Query()
		q.Set("filters", string(filterJSON))
		u.RawQuery = q.Encode()
	}

	resp, err := cli.client.Get(u.String())
	if err != nil {
		return nil, zbxerr.ErrorCannotFetchData.Wrap(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, zbxerr.ErrorCannotFetchData.Wrap(err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr ErrorMessage
		if err = json.Unmarshal(body, &apiErr); err != nil {
			// If we can't parse the error, return the raw body.
			return nil, zbxerr.New(string(body))
		}
		return nil, zbxerr.New(apiErr.Message)
	}

	return body, nil
}

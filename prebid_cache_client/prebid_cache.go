package prebid_cache_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/net/context/ctxhttp"
)

// This file is deprecated, and is only used to cache things for the legacy (/auction) endpoint.
// For /openrtb2/auction cache, see client.go in this package.

type CacheObject struct {
	Value   interface{}
	UUID    string
	IsVideo bool
}

type BidCache struct {
	Adm    string `json:"adm,omitempty"`
	NURL   string `json:"nurl,omitempty"`
	Width  int64  `json:"width,omitempty"`
	Height int64  `json:"height,omitempty"`
}

// internal protocol objects
type putObject struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type putRequest struct {
	Puts []putObject `json:"puts"`
}

type responseObject struct {
	UUID string `json:"uuid"`
}
type response struct {
	Responses []responseObject `json:"responses"`
}

var (
	client  *http.Client
	baseURL string
	putURL  string
)

// InitPrebidCache setup the global prebid cache
func InitPrebidCache(baseurl string) {
	baseURL = baseurl
	putURL = fmt.Sprintf("%s/cache", baseURL)

	ts := &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 65,
	}

	client = &http.Client{
		Transport: ts,
	}
}

// Put will send the array of objs and update each with a UUID
func Put(ctx context.Context, objs []*CacheObject) error {
	// Fixes #197
	if len(objs) == 0 {
		return nil
	}
	pr := putRequest{Puts: make([]putObject, len(objs))}
	for i, obj := range objs {
		if obj.IsVideo {
			pr.Puts[i].Type = "xml"
		} else {
			pr.Puts[i].Type = "json"
		}
		pr.Puts[i].Value = obj.Value
	}
	// Don't want to escape the HTML for adm and nurl
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(pr)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", putURL, buf)
	if err != nil {
		return err
	}
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")

	anResp, err := ctxhttp.Do(ctx, client, httpReq)
	if err != nil {
		return err
	}
	defer anResp.Body.Close()

	if anResp.StatusCode != 200 {
		return fmt.Errorf("HTTP status code %d", anResp.StatusCode)
	}

	var resp response
	if err := json.NewDecoder(anResp.Body).Decode(&resp); err != nil {
		return err
	}

	if len(resp.Responses) != len(objs) {
		return fmt.Errorf("Put response length didn't match")
	}

	for i, r := range resp.Responses {
		objs[i].UUID = r.UUID
	}

	return nil
}

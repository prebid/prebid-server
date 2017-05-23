package prebid_cache_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/net/context/ctxhttp"
)

type CacheObject struct {
	Value string
	UUID  string
}

// internal protocol objects
type putObject struct {
	Value string `json:"value"`
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
	putURL = fmt.Sprintf("%s/put", baseURL)

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
	pr := putRequest{Puts: make([]putObject, len(objs))}
	for i, obj := range objs {
		pr.Puts[i].Value = obj.Value
	}

	reqJSON, err := json.Marshal(pr)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", putURL, bytes.NewBuffer(reqJSON))
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")

	anResp, err := ctxhttp.Do(ctx, client, httpReq)
	if err != nil {
		return err
	}

	if anResp.StatusCode != 200 {
		return fmt.Errorf("HTTP status code %d", anResp.StatusCode)
	}
	defer anResp.Body.Close()

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

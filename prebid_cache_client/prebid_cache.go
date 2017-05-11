package prebid_cache_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/net/context/ctxhttp"
)

type CacheObject struct {
	Key   string
	Value string
	UUID  string
}

// internal protocol objects

type putObject struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type putRequest struct {
	Puts []putObject `json:"puts"`
}

type responseObject struct {
	Key  string `json:"key"`
	UUID string `json:"uuid"`
}
type response struct {
	Responses []responseObject `json:"responses"`
}

var client *http.Client
var base_url string

func InitPrebidCache(baseurl string) {
	base_url = baseurl

	ts := &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 65,
	}

	client = &http.Client{
		Transport: ts,
	}
}

/*
func Get(ctx context.Context, uuid string) string {

}
*/

// will send the array of objs and update each with a UUID
func Put(ctx context.Context, objs []*CacheObject) error {
	pr := putRequest{Puts: make([]putObject, len(objs))}
	for i, obj := range objs {
		pr.Puts[i].Key = obj.Key
		pr.Puts[i].Value = obj.Value
	}

	reqJSON, err := json.Marshal(pr)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/put", base_url), bytes.NewBuffer(reqJSON))
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
	body, err := ioutil.ReadAll(anResp.Body)
	if err != nil {
		return err
	}

	var resp response
	err = json.Unmarshal(body, &resp)
	if err != nil {
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

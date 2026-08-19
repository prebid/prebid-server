package enrichment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

const (
	GDPRConsentHeader = "gdpr-consent"
	maxSnippetBytes   = 1024
)

type Client struct {
	httpClient *http.Client
}

func NewClient(httpClient *http.Client) *Client {
	return &Client{httpClient: httpClient}
}

func (c *Client) Get(ctx context.Context, url, consent string) (Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Response{}, &Error{Kind: KindRequest, Err: err}
	}
	if consent != "" {
		req.Header.Set(GDPRConsentHeader, consent)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		kind := KindTransport
		if errors.Is(err, context.DeadlineExceeded) {
			kind = KindTimeout
		}
		return Response{}, &Error{Kind: kind, Err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	// A non-2xx is a failed resolution: without this check a 429 or 500 carrying a JSON-ish body
	// would be counted as a success.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxSnippetBytes))
		_, _ = io.Copy(io.Discard, resp.Body)
		return Response{}, &Error{
			Kind:            KindStatus,
			Status:          resp.StatusCode,
			Err:             fmt.Errorf("resolution API returned %d", resp.StatusCode),
			ResponseSnippet: strings.Join(strings.Fields(string(body)), " "),
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, &Error{Kind: KindBodyRead, Status: resp.StatusCode, Err: err}
	}

	var parsed Response
	if err = jsonutil.Unmarshal(body, &parsed); err != nil {
		return Response{}, &Error{Kind: KindParse, Status: resp.StatusCode, Err: err}
	}
	parsed.Status = resp.StatusCode
	return parsed, nil
}

// Response is the identity-resolution S2S response. IntentIQ returns it as an object on a
// hit but as an empty string ("") on an empty/invalid response, so a non-object
// data is treated as absent rather than failing the whole parse. Eids applies that leniency.
type Response struct {
	Data       json.RawMessage `json:"data"`
	Cttl       *int64          `json:"cttl"`
	AbTestUUID string          `json:"abTestUuid"`
	Tc         *int64          `json:"tc"`

	// Status is the HTTP status the response, used for debug tracing
	Status int `json:"-"`
}

func (r Response) Eids() []openrtb2.EID {
	var d struct {
		Eids []openrtb2.EID `json:"eids"`
	}
	if err := json.Unmarshal(r.Data, &d); err != nil {
		return nil
	}
	return d.Eids
}

func (r Response) CTTL() time.Duration {
	if r.Cttl == nil {
		return 0
	}
	return time.Duration(*r.Cttl) * time.Second
}

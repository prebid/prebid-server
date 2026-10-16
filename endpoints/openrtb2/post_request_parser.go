package openrtb2

import (
	"fmt"
	"io"
	"net/http"

	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/util/httputil"
)

// readRequestBody reads the raw OpenRTB JSON from a POST request body. It applies
// content-encoding negotiation and enforces the configured maximum request size.
//
// These concerns are specific to a real HTTP payload, which is why they live here
// rather than in the shared parseRequest flow. GET requests build their JSON from
// query parameters (see get_request_parser.go) and bypass this logic entirely.
func readRequestBody(httpRequest *http.Request, cfg *config.Configuration) ([]byte, error) {
	var err error
	var r io.ReadCloser = httpRequest.Body

	reqContentEncoding := httputil.ContentEncoding(httpRequest.Header.Get("Content-Encoding"))
	if reqContentEncoding != "" {
		if !cfg.Compression.Request.IsSupported(reqContentEncoding) {
			return nil, fmt.Errorf("Content-Encoding of type %s is not supported", reqContentEncoding)
		}
		r, err = getCompressionEnabledReader(httpRequest.Body, reqContentEncoding)
		if err != nil {
			return nil, err
		}
	}
	defer r.Close()

	limitedReqReader := &io.LimitedReader{
		R: r,
		N: cfg.MaxRequestSize,
	}

	requestJson, err := io.ReadAll(limitedReqReader)
	if err != nil {
		return nil, err
	}

	if limitedReqReader.N <= 0 {
		// Limited Reader returns 0 if the request was exactly at the max size or over the limit.
		// This is because it only reads up to N bytes. To check if the request was too large,
		//  we need to look at the next byte of its underlying reader, limitedReader.R.
		if _, err := limitedReqReader.R.Read(make([]byte, 1)); err != io.EOF {
			// Discard the rest of the request body so that the connection can be reused.
			io.Copy(io.Discard, httpRequest.Body)
			return nil, fmt.Errorf("request size exceeded max size of %d bytes.", cfg.MaxRequestSize)
		}
	}

	return requestJson, nil
}

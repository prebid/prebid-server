#!/usr/bin/env python3
"""Local stand-in for the IntentIQ S2S resolve (at=39) and reporting (at=45) endpoints."""
import json
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlparse

PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 9999

RESOLVE_BODY = {
    "data": {"eids": [{"source": "intentiq.com", "uids": [{"id": "mock-iiq-id", "atype": 1}]}]},
    "cttl": 600000,
    "abTestUuid": "0123456789abcdefghijklmnopqrstuvwxyz",
    "tc": 41,
}


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        query = parse_qs(urlparse(self.path).query)
        at = query.get("at", [""])[0]
        if at == "39":
            print(f"[resolve] dpi={query.get('dpi')} ip={query.get('ip')} consent={self.headers.get('gdpr-consent')}", flush=True)
            body = json.dumps(RESOLVE_BODY).encode()
        else:
            rdata = json.loads(query.get("rdata", ["{}"])[0])
            print(f"[report] {json.dumps(rdata)}", flush=True)
            body = b""
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *_):
        pass


ThreadingHTTPServer(("127.0.0.1", PORT), Handler).serve_forever()

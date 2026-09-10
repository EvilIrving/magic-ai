#!/usr/bin/env python3
import json
import os
import urllib.error
import urllib.request
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path


LISTEN = ("127.0.0.1", 18765)
UPSTREAM = "https://api.shu.cool"
OUTPUT = Path(__file__).resolve().parent.parent / "gateway-agent-packaging-request.json"


class Handler(BaseHTTPRequestHandler):
    def forward(self, raw_request=None):
        method = self.command
        try:
            request_body = json.loads(raw_request) if raw_request else None
        except json.JSONDecodeError:
            request_body = None
        api_key = os.environ["SHU_API_KEY"]
        headers = {
            "Authorization": f"Bearer {api_key}",
            "Accept": self.headers.get("accept", "application/json"),
            "Content-Type": self.headers.get("content-type", "application/json"),
            "User-Agent": self.headers.get("user-agent", "pi-request-capture"),
        }
        upstream_request = urllib.request.Request(
            UPSTREAM + self.path,
            data=raw_request,
            headers=headers,
            method=method,
        )
        try:
            upstream_response = urllib.request.urlopen(upstream_request, timeout=300)
        except urllib.error.HTTPError as error:
            upstream_response = error
        raw_response = upstream_response.read()
        content_type = upstream_response.headers.get("content-type", "application/octet-stream")
        self.send_response(upstream_response.status)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(raw_response)))
        self.end_headers()
        self.wfile.write(raw_response)
        if method != "POST":
            return
        response_text = raw_response.decode("utf-8", errors="replace")
        try:
            response_body = json.loads(response_text)
        except json.JSONDecodeError:
            response_body = None
        report = {
            "captured_at": datetime.now(timezone.utc).isoformat(),
            "description": "Pi 0.81.1 实际发出的代理请求，认证信息已脱敏。",
            "request": {
                "method": "POST",
                "url": UPSTREAM + self.path,
                "headers": {
                    "Accept": headers["Accept"],
                    "Content-Type": headers["Content-Type"],
                    "User-Agent": headers["User-Agent"],
                    "Authorization": "Bearer（已脱敏）",
                },
                "body": request_body,
                "raw_text": raw_request.decode("utf-8", errors="replace"),
            },
            "response": {
                "status": upstream_response.status,
                "headers": {
                    name.lower(): value
                    for name, value in upstream_response.headers.items()
                    if name.lower() in {"content-type", "x-request-id", "request-id"}
                },
                "body": response_body,
                "raw_text": response_text,
            },
        }
        OUTPUT.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    def do_GET(self):
        self.forward()

    def do_POST(self):
        length = int(self.headers.get("content-length", "0"))
        self.forward(self.rfile.read(length))

    def log_message(self, format, *args):
        return


if __name__ == "__main__":
    server = ThreadingHTTPServer(LISTEN, Handler)
    print(f"http://{LISTEN[0]}:{LISTEN[1]}", flush=True)
    server.serve_forever()

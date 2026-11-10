#!/usr/bin/env python3
"""Docker falso para o teste de navegador: responde como o docker-socket-proxy. Uso: docker-falso.py <porta>"""
import json
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer

estado = {"cache": 14_642_571_481}


class Docker(BaseHTTPRequestHandler):
    def responder(self, corpo, status=200):
        b = json.dumps(corpo).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)

    def do_GET(self):
        if self.path == "/v1.47/system/df":
            return self.responder({
                "LayersSize": 1_946_260_943,
                "Images": [{"RepoTags": ["acervo:abc"], "Size": 902_410_231, "SharedSize": 0, "Containers": 1}],
                "Containers": [
                    {"Id": "caddy", "Labels": {"com.docker.compose.project": "proxy"}, "Names": ["/caddy"], "Image": "caddy:2-alpine",
                     "State": "running", "Status": "Up 5 hours", "SizeRw": 4096},
                    {"Id": "acervo", "Labels": {"com.docker.compose.project": "acervo"}, "Names": ["/acervo-acervo-1"], "Image": "acervo:abc",
                     "State": "running", "Status": "Up 2 days", "SizeRw": 0},
                    {"Id": "whisper", "Labels": {"com.docker.compose.project": "whisper"}, "Names": ["/whisper"], "Image": "whisper:abc",
                     "State": "running", "Status": "Up 1 hour", "SizeRw": 0},
                ],
                "BuildCache": [{"Size": estado["cache"], "InUse": False, "Shared": False}],
            })
        self.responder({"message": "proibido"}, 403)

    def do_POST(self):
        if self.path.startswith("/v1.47/build/prune"):
            liberado, estado["cache"] = estado["cache"], 0
            return self.responder({"SpaceReclaimed": liberado})
        if self.path == "/v1.47/images/prune":
            return self.responder({"SpaceReclaimed": 0})
        self.responder({"message": "proibido"}, 403)

    def log_message(self, *args):
        pass


HTTPServer(("127.0.0.1", int(sys.argv[1])), Docker).serve_forever()

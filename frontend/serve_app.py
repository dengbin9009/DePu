from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import unquote
from urllib.request import Request, urlopen
from urllib.error import HTTPError
import mimetypes
import os

ROOT = Path(__file__).resolve().parent / 'dist'
API_BASE = 'http://127.0.0.1:5174'

def resolve_spa_file(path, root=ROOT):
    decoded_path = unquote(path)
    file_path = root / decoded_path.lstrip('/')
    if decoded_path == '/' or not file_path.exists() or file_path.is_dir():
        return root / 'index.html'
    return file_path

class AppHandler(BaseHTTPRequestHandler):
    def do_HEAD(self):
        self.handle_request(head_only=True)

    def do_GET(self):
        self.handle_request(head_only=False)

    def do_POST(self):
        self.handle_request(head_only=False)

    def do_PATCH(self):
        self.handle_request(head_only=False)

    def do_DELETE(self):
        self.handle_request(head_only=False)

    def handle_request(self, head_only=False):
        path = self.path.split('?', 1)[0]
        if path.startswith('/api/') or path == '/health':
            self.proxy_api(head_only)
            return
        self.serve_spa(path, head_only)

    def proxy_api(self, head_only=False):
        target = API_BASE + self.path
        body = None
        length = int(self.headers.get('Content-Length', '0') or '0')
        if length:
            body = self.rfile.read(length)
        headers = {k: v for k, v in self.headers.items() if k.lower() not in {'host', 'content-length', 'connection'}}
        req = Request(target, data=body, headers=headers, method=self.command)
        try:
            with urlopen(req, timeout=20) as resp:
                payload = resp.read()
                self.send_response(resp.status)
                for k, v in resp.getheaders():
                    if k.lower() in {'transfer-encoding', 'connection', 'server', 'date'}:
                        continue
                    self.send_header(k, v)
                self.end_headers()
                if not head_only:
                    self.wfile.write(payload)
        except HTTPError as e:
            payload = e.read()
            self.send_response(e.code)
            for k, v in e.headers.items():
                if k.lower() in {'transfer-encoding', 'connection', 'server', 'date'}:
                    continue
                self.send_header(k, v)
            self.end_headers()
            if not head_only:
                self.wfile.write(payload)

    def serve_spa(self, path, head_only=False):
        file_path = resolve_spa_file(path)
        ctype = mimetypes.guess_type(str(file_path))[0] or 'application/octet-stream'
        data = file_path.read_bytes()
        self.send_response(200)
        self.send_header('Content-Type', ctype)
        self.send_header('Content-Length', str(len(data)))
        self.end_headers()
        if not head_only:
            self.wfile.write(data)

if __name__ == '__main__':
    os.chdir(ROOT)
    server = ThreadingHTTPServer(('0.0.0.0', 5175), AppHandler)
    print('Serving app on http://0.0.0.0:5175')
    server.serve_forever()

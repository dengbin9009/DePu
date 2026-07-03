from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import unquote
import os

ROOT = Path(__file__).resolve().parent / 'dist'
os.chdir(ROOT)

def resolve_spa_file(path, root=ROOT):
    decoded_path = unquote(path)
    target = root / decoded_path.lstrip('/')
    if decoded_path == '/' or target.exists():
        return target
    return root / 'index.html'

class SPAHandler(SimpleHTTPRequestHandler):
    def do_GET(self):
        path = self.path.split('?', 1)[0].split('#', 1)[0]
        if path.startswith('/api/'):
            self.send_error(404, 'API should be served by backend')
            return
        target = resolve_spa_file(path)
        if target != ROOT / 'index.html' or path == '/':
            return super().do_GET()
        self.path = '/index.html'
        return super().do_GET()

    def do_HEAD(self):
        path = self.path.split('?', 1)[0].split('#', 1)[0]
        if path.startswith('/api/'):
            self.send_error(404, 'API should be served by backend')
            return
        target = resolve_spa_file(path)
        if target != ROOT / 'index.html' or path == '/':
            return super().do_HEAD()
        self.path = '/index.html'
        return super().do_HEAD()

if __name__ == '__main__':
    port = 5175
    server = ThreadingHTTPServer(('0.0.0.0', port), SPAHandler)
    print(f'Serving SPA on http://0.0.0.0:{port}')
    server.serve_forever()

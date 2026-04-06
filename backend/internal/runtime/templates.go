package runtime

import (
	"fmt"
	"strings"
)

// GenerateDockerfileWithBase creates a Dockerfile that uses a pre-built Applad
// base image (applad-base-{runtime}) instead of the upstream image. The base
// image already contains the HTTP wrapper, so the Dockerfile only needs to COPY
// the user's source file. Falls back to GenerateDockerfile if no base image
// exists for the given runtime.
func GenerateDockerfileWithBase(runtime, entrypoint, source string) string {
	baseName := GetBaseImageName(runtime)
	if baseName == "" {
		return GenerateDockerfile(runtime, entrypoint, source)
	}

	filename := sourceFilename(runtime, entrypoint)

	switch {
	case strings.HasPrefix(runtime, "node"):
		// If source has its own server, use the standard Dockerfile
		if strings.Contains(source, "listen(") || strings.Contains(source, "createServer") {
			return GenerateDockerfile(runtime, entrypoint, source)
		}
		return fmt.Sprintf(`FROM %s
COPY %s /app/handler.js
CMD ["node", "wrapper.js"]
`, baseName, filename)

	case strings.HasPrefix(runtime, "bun"):
		if strings.Contains(source, "serve") || strings.Contains(source, "listen") {
			return GenerateDockerfile(runtime, entrypoint, source)
		}
		return fmt.Sprintf(`FROM %s
COPY %s /app/handler.ts
CMD ["bun", "run", "wrapper.ts"]
`, baseName, filename)

	case strings.HasPrefix(runtime, "python"):
		if strings.Contains(source, "Flask") || strings.Contains(source, "uvicorn") || strings.Contains(source, "HTTPServer") {
			return GenerateDockerfile(runtime, entrypoint, source)
		}
		return fmt.Sprintf(`FROM %s
COPY %s /app/handler.py
CMD ["python", "wrapper.py"]
`, baseName, filename)

	case strings.HasPrefix(runtime, "go"):
		if strings.Contains(source, "ListenAndServe") || strings.Contains(source, "http.Handle") {
			return GenerateDockerfile(runtime, entrypoint, source)
		}
		return fmt.Sprintf(`FROM %s
COPY %s /app/handler.go
RUN go build -o fn wrapper.go handler.go
CMD ["/app/fn"]
`, baseName, filename)

	case strings.HasPrefix(runtime, "dart"):
		if strings.Contains(source, "HttpServer") || strings.Contains(source, "shelf") {
			return GenerateDockerfile(runtime, entrypoint, source)
		}
		return fmt.Sprintf(`FROM %s
COPY %s /app/handler.dart
CMD ["dart", "run", "wrapper.dart"]
`, baseName, filename)

	case strings.HasPrefix(runtime, "ruby"):
		return fmt.Sprintf(`FROM %s
COPY %s /app/handler.rb
CMD ["ruby", "wrapper.rb"]
`, baseName, filename)

	case strings.HasPrefix(runtime, "php"):
		return fmt.Sprintf(`FROM %s
COPY %s /app/handler.php
CMD ["php", "wrapper.php"]
`, baseName, filename)

	default:
		return GenerateDockerfile(runtime, entrypoint, source)
	}
}

// GenerateDockerfile creates a Dockerfile for the given runtime and source.
// Returns empty string if the runtime is not recognized.
func GenerateDockerfile(runtime, entrypoint, source string) string {
	switch {
	case strings.HasPrefix(runtime, "node"):
		return dockerfileNode(runtime, source)
	case strings.HasPrefix(runtime, "bun"):
		return dockerfileBun(source)
	case strings.HasPrefix(runtime, "python"):
		return dockerfilePython(source)
	case strings.HasPrefix(runtime, "go"):
		return dockerfileGo(source)
	case strings.HasPrefix(runtime, "dart"):
		return dockerfileDart(source)
	case strings.HasPrefix(runtime, "rust"):
		return dockerfileRust(source)
	case strings.HasPrefix(runtime, "ruby"):
		return dockerfileRuby(source)
	case strings.HasPrefix(runtime, "php"):
		return dockerfilePHP(source)
	case runtime == "custom":
		// Custom runtime — user must provide a Dockerfile via the dockerfile field
		return ""
	default:
		return ""
	}
}

// SupportedRuntimes returns the list of built-in runtimes.
func SupportedRuntimes() []map[string]string {
	return []map[string]string{
		{"id": "node-18", "name": "Node.js 18", "image": "node:18-alpine"},
		{"id": "node-20", "name": "Node.js 20", "image": "node:20-alpine"},
		{"id": "node-22", "name": "Node.js 22", "image": "node:22-alpine"},
		{"id": "bun-1", "name": "Bun 1.x", "image": "oven/bun:alpine"},
		{"id": "python-3.11", "name": "Python 3.11", "image": "python:3.11-alpine"},
		{"id": "python-3.12", "name": "Python 3.12", "image": "python:3.12-alpine"},
		{"id": "go-1.22", "name": "Go 1.22", "image": "golang:1.22-alpine"},
		{"id": "dart-3", "name": "Dart 3.x", "image": "dart:stable"},
		{"id": "rust-1", "name": "Rust 1.x", "image": "rust:alpine"},
		{"id": "ruby-3", "name": "Ruby 3.x", "image": "ruby:3-alpine"},
		{"id": "php-8", "name": "PHP 8.x", "image": "php:8-alpine"},
		{"id": "custom", "name": "Custom Dockerfile", "image": ""},
	}
}

func nodeVersion(runtime string) string {
	switch runtime {
	case "node-20":
		return "20"
	case "node-22":
		return "22"
	default:
		return "18"
	}
}

// --- Dockerfile templates ---
// Each template wraps user source code in a minimal HTTP server that:
//   - Listens on port 3000
//   - POST / receives invocation payload as JSON body
//   - Returns the function output as the response

func dockerfileNode(runtime, source string) string {
	version := nodeVersion(runtime)
	// If source already contains an HTTP server (express/http), just run it
	if strings.Contains(source, "listen(") || strings.Contains(source, "createServer") {
		return fmt.Sprintf(`FROM node:%s-alpine
WORKDIR /app
COPY index.js .
RUN if [ -f package.json ]; then npm install; fi
EXPOSE 3000
CMD ["node", "index.js"]
`, version)
	}

	// Wrap in a minimal HTTP server
	return fmt.Sprintf(`FROM node:%s-alpine
WORKDIR /app
COPY index.js ./handler.js
RUN cat > index.js << 'WRAPPER'
const http = require('http');
const handler = require('./handler.js');
const fn = typeof handler === 'function' ? handler : handler.default || handler.handler || handler.main || (() => ({}));
const server = http.createServer(async (req, res) => {
  if (req.method === 'GET') { res.writeHead(200); res.end('ok'); return; }
  let body = '';
  req.on('data', c => body += c);
  req.on('end', async () => {
    try {
      const payload = body ? JSON.parse(body) : {};
      const result = await fn(payload);
      res.writeHead(200, {'Content-Type': 'application/json'});
      res.end(JSON.stringify(result ?? {}));
    } catch(e) {
      res.writeHead(500, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({error: e.message}));
    }
  });
});
server.listen(3000, () => console.log('ready'));
WRAPPER
EXPOSE 3000
CMD ["node", "index.js"]
`, version)
}

func dockerfileBun(source string) string {
	if strings.Contains(source, "serve") || strings.Contains(source, "listen") {
		return `FROM oven/bun:alpine
WORKDIR /app
COPY index.ts .
EXPOSE 3000
CMD ["bun", "run", "index.ts"]
`
	}

	return `FROM oven/bun:alpine
WORKDIR /app
COPY index.ts ./handler.ts
RUN cat > index.ts << 'WRAPPER'
import handler from './handler.ts';
const fn = typeof handler === 'function' ? handler : handler.default || handler.handler || (() => ({}));
Bun.serve({
  port: 3000,
  async fetch(req) {
    if (req.method === 'GET') return new Response('ok');
    try {
      const payload = await req.json().catch(() => ({}));
      const result = await fn(payload);
      return Response.json(result ?? {});
    } catch(e) {
      return Response.json({error: e.message}, {status: 500});
    }
  },
});
console.log('ready');
WRAPPER
EXPOSE 3000
CMD ["bun", "run", "index.ts"]
`
}

func dockerfilePython(source string) string {
	if strings.Contains(source, "Flask") || strings.Contains(source, "uvicorn") || strings.Contains(source, "HTTPServer") {
		return `FROM python:3.12-alpine
WORKDIR /app
COPY main.py .
RUN if [ -f requirements.txt ]; then pip install -r requirements.txt; fi
EXPOSE 3000
CMD ["python", "main.py"]
`
	}

	return `FROM python:3.12-alpine
WORKDIR /app
COPY main.py ./handler.py
RUN cat > main.py << 'WRAPPER'
import json, importlib.util, sys
from http.server import HTTPServer, BaseHTTPRequestHandler
spec = importlib.util.spec_from_file_location("handler", "/app/handler.py")
mod = importlib.util.module_from_spec(spec)
spec.loader.exec_module(mod)
fn = getattr(mod, 'handler', getattr(mod, 'main', getattr(mod, 'default', None)))
if fn is None:
    for name in dir(mod):
        obj = getattr(mod, name)
        if callable(obj) and not name.startswith('_'):
            fn = obj; break
if fn is None: fn = lambda x: {}
class H(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200); self.end_headers(); self.wfile.write(b'ok')
    def do_POST(self):
        length = int(self.headers.get('Content-Length', 0))
        body = json.loads(self.rfile.read(length)) if length > 0 else {}
        try:
            result = fn(body)
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps(result or {}).encode())
        except Exception as e:
            self.send_response(500)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps({'error': str(e)}).encode())
    def log_message(self, *a): pass
print('ready')
HTTPServer(('', 3000), H).serve_forever()
WRAPPER
EXPOSE 3000
CMD ["python", "main.py"]
`
}

func dockerfileGo(source string) string {
	if strings.Contains(source, "ListenAndServe") || strings.Contains(source, "http.Handle") {
		return `FROM golang:1.22-alpine AS build
WORKDIR /app
COPY main.go .
RUN go build -o handler main.go
FROM alpine:latest
COPY --from=build /app/handler /handler
EXPOSE 3000
CMD ["/handler"]
`
	}

	return `FROM golang:1.22-alpine AS build
WORKDIR /app
COPY main.go ./handler.go
RUN cat > main.go << 'WRAPPER'
package main

import (
	"encoding/json"
	"io"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Write([]byte("ok"))
			return
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		json.Unmarshal(body, &payload)
		result := handler(payload)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})
	http.ListenAndServe(":3000", nil)
}
WRAPPER
RUN go build -o fn main.go handler.go
FROM alpine:latest
COPY --from=build /app/fn /fn
EXPOSE 3000
CMD ["/fn"]
`
}

func dockerfileDart(source string) string {
	if strings.Contains(source, "HttpServer") || strings.Contains(source, "shelf") {
		return `FROM dart:stable AS build
WORKDIR /app
COPY main.dart .
RUN dart compile exe main.dart -o handler
FROM alpine:latest
COPY --from=build /app/handler /handler
COPY --from=build /runtime/ /runtime/
EXPOSE 3000
CMD ["/handler"]
`
	}

	return `FROM dart:stable AS build
WORKDIR /app
COPY main.dart ./handler.dart
RUN cat > main.dart << 'WRAPPER'
import 'dart:convert';
import 'dart:io';
import 'handler.dart' as handler;

void main() async {
  final server = await HttpServer.bind('0.0.0.0', 3000);
  print('ready');
  await for (final req in server) {
    if (req.method == 'GET') {
      req.response..write('ok')..close();
      continue;
    }
    try {
      final body = await utf8.decoder.bind(req).join();
      final payload = body.isNotEmpty ? jsonDecode(body) : {};
      final result = await handler.main(payload);
      req.response
        ..headers.contentType = ContentType.json
        ..write(jsonEncode(result ?? {}))
        ..close();
    } catch (e) {
      req.response
        ..statusCode = 500
        ..headers.contentType = ContentType.json
        ..write(jsonEncode({'error': e.toString()}))
        ..close();
    }
  }
}
WRAPPER
RUN dart compile exe main.dart -o fn 2>/dev/null || echo "compile with source"
EXPOSE 3000
CMD ["dart", "run", "main.dart"]
`
}

func dockerfileRust(source string) string {
	return `FROM rust:alpine AS build
RUN apk add musl-dev
WORKDIR /app
COPY main.rs ./src/main.rs
RUN cat > Cargo.toml << 'EOF'
[package]
name = "fn"
version = "0.1.0"
edition = "2021"
EOF
RUN cargo build --release 2>/dev/null || true
EXPOSE 3000
CMD ["cargo", "run", "--release"]
`
}

func dockerfileRuby(source string) string {
	return `FROM ruby:3-alpine
WORKDIR /app
COPY main.py ./handler.rb
RUN cat > main.rb << 'WRAPPER'
require 'webrick'
require 'json'
load '/app/handler.rb'
fn = method(:handler) rescue method(:main) rescue proc { |x| {} }
server = WEBrick::HTTPServer.new(Port: 3000, Logger: WEBrick::Log.new("/dev/null"))
server.mount_proc '/' do |req, res|
  if req.request_method == 'GET'
    res.body = 'ok'
  else
    payload = JSON.parse(req.body || '{}')
    result = fn.call(payload)
    res['Content-Type'] = 'application/json'
    res.body = JSON.generate(result || {})
  end
end
puts 'ready'
server.start
WRAPPER
EXPOSE 3000
CMD ["ruby", "main.rb"]
`
}

func dockerfilePHP(source string) string {
	return `FROM php:8-alpine
WORKDIR /app
COPY main.py ./handler.php
RUN cat > main.php << 'WRAPPER'
<?php
$handler = require '/app/handler.php';
$socket = stream_socket_server('tcp://0.0.0.0:3000');
echo "ready\n";
while ($conn = stream_socket_accept($socket, -1)) {
    $req = fread($conn, 65536);
    $lines = explode("\r\n", $req);
    $method = explode(' ', $lines[0])[0] ?? 'GET';
    if ($method === 'GET') {
        fwrite($conn, "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok");
    } else {
        $body = substr($req, strpos($req, "\r\n\r\n") + 4);
        $payload = json_decode($body, true) ?? [];
        try {
            $result = is_callable($handler) ? $handler($payload) : [];
            $json = json_encode($result ?? []);
            fwrite($conn, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: ".strlen($json)."\r\n\r\n".$json);
        } catch (Exception $e) {
            $json = json_encode(['error' => $e->getMessage()]);
            fwrite($conn, "HTTP/1.1 500 Internal Server Error\r\nContent-Type: application/json\r\nContent-Length: ".strlen($json)."\r\n\r\n".$json);
        }
    }
    fclose($conn);
}
WRAPPER
EXPOSE 3000
CMD ["php", "main.php"]
`
}

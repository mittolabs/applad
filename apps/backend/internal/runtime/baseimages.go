package runtime

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// BaseImage defines a pre-built base image with the HTTP wrapper pre-installed.
type BaseImage struct {
	Runtime string // runtime identifier (e.g., "node-18")
	Name    string // base image tag (e.g., "applad-base-node:18")
	From    string // upstream image (e.g., "node:18-alpine")
}

// baseImages lists all pre-built base images that Applad manages.
var baseImages = []BaseImage{
	{Runtime: "node-18", Name: "applad-base-node:18", From: "node:18-alpine"},
	{Runtime: "node-20", Name: "applad-base-node:20", From: "node:20-alpine"},
	{Runtime: "node-22", Name: "applad-base-node:22", From: "node:22-alpine"},
	{Runtime: "bun-1", Name: "applad-base-bun:1", From: "oven/bun:alpine"},
	{Runtime: "python-3.11", Name: "applad-base-python:3.11", From: "python:3.11-alpine"},
	{Runtime: "python-3.12", Name: "applad-base-python:3.12", From: "python:3.12-alpine"},
	{Runtime: "go-1.22", Name: "applad-base-go:1.22", From: "golang:1.22-alpine"},
	{Runtime: "dart-3", Name: "applad-base-dart:3", From: "dart:stable"},
	{Runtime: "rust-1", Name: "applad-base-rust:1", From: "rust:alpine"},
	{Runtime: "ruby-3", Name: "applad-base-ruby:3", From: "ruby:3-alpine"},
	{Runtime: "php-8", Name: "applad-base-php:8", From: "php:8-alpine"},
}

// EnsureBaseImages checks if all Applad base images exist and builds any that
// are missing. This should be called at worker startup to pre-warm the image
// cache so function builds only need to COPY the user's source file.
func EnsureBaseImages(ctx context.Context, docker *Client) {
	for _, bi := range baseImages {
		if imageExists(ctx, docker, bi.Name) {
			log.Printf("baseimages: %s already exists", bi.Name)
			continue
		}
		log.Printf("baseimages: building %s from %s", bi.Name, bi.From)
		if err := BuildBaseImage(ctx, docker, bi.Runtime); err != nil {
			log.Printf("baseimages: failed to build %s: %v", bi.Name, err)
		} else {
			log.Printf("baseimages: %s ready", bi.Name)
		}
	}
}

// BuildBaseImage builds a single base image for a given runtime. The base image
// includes the upstream runtime plus the HTTP wrapper script pre-installed, so
// function builds only need to COPY the user source file into /app.
func BuildBaseImage(ctx context.Context, docker *Client, runtimeID string) error {
	bi := findBaseImage(runtimeID)
	if bi == nil {
		return fmt.Errorf("baseimages: unknown runtime %q", runtimeID)
	}

	dockerfile := generateBaseDockerfile(bi)
	if dockerfile == "" {
		return fmt.Errorf("baseimages: no base dockerfile for runtime %q", runtimeID)
	}

	// Create tar context
	tarBuf := new(bytes.Buffer)
	tw := tar.NewWriter(tarBuf)
	addToTar(tw, "Dockerfile", []byte(dockerfile))
	tw.Close()

	_, buildErr := docker.BuildImage(ctx, bi.Name, tarBuf)
	return buildErr
}

// GetBaseImageName returns the base image name for a runtime, or empty string
// if no base image is defined for that runtime.
func GetBaseImageName(runtimeID string) string {
	bi := findBaseImage(runtimeID)
	if bi == nil {
		return ""
	}
	return bi.Name
}

// findBaseImage looks up a BaseImage by runtime ID.
func findBaseImage(runtimeID string) *BaseImage {
	for i := range baseImages {
		if baseImages[i].Runtime == runtimeID {
			return &baseImages[i]
		}
	}
	return nil
}

// imageExists checks if a Docker image exists locally.
func imageExists(ctx context.Context, docker *Client, imageName string) bool {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf(docker.baseURL+"/v1.43/images/%s/json", imageName), nil)
	if err != nil {
		return false
	}

	resp, err := docker.httpClient.Do(req)
	if err != nil {
		return false
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// generateBaseDockerfile creates a Dockerfile for a base image that pre-installs
// the HTTP wrapper. User builds then just do: FROM applad-base-{runtime} + COPY source.
func generateBaseDockerfile(bi *BaseImage) string {
	switch {
	case strings.HasPrefix(bi.Runtime, "node"):
		return fmt.Sprintf(`FROM %s
WORKDIR /app
RUN cat > /app/wrapper.js << 'WRAPPER'
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
CMD ["node", "wrapper.js"]
`, bi.From)

	case strings.HasPrefix(bi.Runtime, "bun"):
		return fmt.Sprintf(`FROM %s
WORKDIR /app
RUN cat > /app/wrapper.ts << 'WRAPPER'
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
CMD ["bun", "run", "wrapper.ts"]
`, bi.From)

	case strings.HasPrefix(bi.Runtime, "python"):
		return fmt.Sprintf(`FROM %s
WORKDIR /app
RUN cat > /app/wrapper.py << 'WRAPPER'
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
CMD ["python", "wrapper.py"]
`, bi.From)

	case strings.HasPrefix(bi.Runtime, "go"):
		return fmt.Sprintf(`FROM %s
WORKDIR /app
RUN cat > /app/wrapper.go << 'WRAPPER'
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
EXPOSE 3000
`, bi.From)

	case strings.HasPrefix(bi.Runtime, "ruby"):
		return fmt.Sprintf(`FROM %s
WORKDIR /app
RUN cat > /app/wrapper.rb << 'WRAPPER'
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
CMD ["ruby", "wrapper.rb"]
`, bi.From)

	case strings.HasPrefix(bi.Runtime, "php"):
		return fmt.Sprintf(`FROM %s
WORKDIR /app
RUN cat > /app/wrapper.php << 'WRAPPER'
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
CMD ["php", "wrapper.php"]
`, bi.From)

	case strings.HasPrefix(bi.Runtime, "dart"):
		return fmt.Sprintf(`FROM %s
WORKDIR /app
RUN cat > /app/wrapper.dart << 'WRAPPER'
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
EXPOSE 3000
CMD ["dart", "run", "wrapper.dart"]
`, bi.From)

	case strings.HasPrefix(bi.Runtime, "rust"):
		return fmt.Sprintf(`FROM %s
RUN apk add musl-dev
WORKDIR /app
EXPOSE 3000
`, bi.From)

	default:
		return ""
	}
}

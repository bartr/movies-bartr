package httpapi

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
)

// Swagger / OpenAPI assets and routes.
//
// The OpenAPI 3 document is hand-authored at openapi.json and embedded into
// the binary at compile time. The Swagger UI page is a single self-contained
// HTML document that loads swagger-ui-dist from a pinned CDN. The service
// itself has no runtime network dependency.

//go:embed openapi.json
var openAPIDocBytes []byte

// openAPIDoc is openAPIDocBytes minified once at init so the response is a
// stable byte stream and we never re-marshal per request.
var openAPIDoc = func() []byte {
	var buf bytes.Buffer
	if err := json.Compact(&buf, openAPIDocBytes); err != nil {
		// Compile-time JSON; failure here means the embedded asset is bad
		// and we'd rather panic than ship a broken /swagger.json.
		panic("httpapi: invalid embedded openapi.json: " + err.Error())
	}
	return buf.Bytes()
}()

// swaggerIndex is the Swagger UI shell. The CDN URLs are pinned and
// fetched by the browser, not the server.
const swaggerIndex = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Movies API — Swagger UI</title>
  <meta name="robots" content="noindex,nofollow">
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css"
        crossorigin="anonymous"
        referrerpolicy="no-referrer">
  <style>html,body{margin:0;background:#fafafa;}</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"
          crossorigin="anonymous"
          referrerpolicy="no-referrer"></script>
  <script>
    window.addEventListener('load', function () {
      window.ui = SwaggerUIBundle({
        url: '/swagger/v1/swagger.json',
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis]
      });
    });
  </script>
</body>
</html>
`

const robotsTxt = "User-agent: *\nDisallow: /\n"

// rootRedirectHandler permanently redirects "/" to "/swagger".
// Spec §6 routes "/" → "/swagger".
func rootRedirectHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only redirect on exact "/" — chi uses NotFoundHandler for unknown
		// paths, so this handler only fires on the root.
		http.Redirect(w, r, "/swagger", http.StatusMovedPermanently)
	}
}

// swaggerUIHandler serves the Swagger UI HTML shell.
func swaggerUIHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = io.WriteString(w, swaggerIndex)
	}
}

// swaggerJSONHandler serves the embedded OpenAPI 3 document.
func swaggerJSONHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(openAPIDoc)
	}
}

// robotsHandler returns a "disallow everything" robots.txt. The service is
// a local experiment harness, not a public site.
func robotsHandler() http.HandlerFunc {
	body := []byte(robotsTxt)
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", plainTextUTF8)
		_, _ = w.Write(body)
	}
}

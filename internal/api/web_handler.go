package api

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/valyala/fasthttp"
)

// Serve static files (CSS, JS, images)
func (h *Handler) handleStatic(ctx *fasthttp.RequestCtx) {
	filePath := string(ctx.Path())
	// Remove the leading "/static/" from the path
	filePath = strings.TrimPrefix(filePath, "/static/")
	// Build the actual file path
	fullPath := filepath.Join("web/static", filePath)

	// Try to read the file
	content, err := ioutil.ReadFile(fullPath)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("File not found")
		return
	}

	// Set content type based on file extension
	ext := filepath.Ext(filePath)
	switch ext {
	case ".css":
		ctx.SetContentType("text/css")
	case ".js":
		ctx.SetContentType("application/javascript")
	case ".png":
		ctx.SetContentType("image/png")
	case ".jpg", ".jpeg":
		ctx.SetContentType("image/jpeg")
	case ".svg":
		ctx.SetContentType("image/svg+xml")
	default:
		ctx.SetContentType("application/octet-stream")
	}

	// Set the file content
	ctx.SetBody(content)
}

// Serve the main HTML page
func (h *Handler) handleIndex(ctx *fasthttp.RequestCtx) {
	// Read the HTML template
	content, err := ioutil.ReadFile("web/templates/index.html")
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(fmt.Sprintf("Error reading template: %v", err))
		return
	}

	ctx.SetContentType("text/html; charset=utf-8")
	ctx.SetBody(content)
}

// Package pdf handles branded PDF generation via headless Chrome (os/exec).
// PDF generation ALWAYS runs in an asynq worker goroutine — never in an HTTP handler.
package pdf

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"

	"school-platform/internal/config"
)

// Generator handles PDF generation using headless Chrome via os/exec.
type Generator struct {
	cfg         *config.Config
	templateDir string
}

// NewGenerator creates a new PDF Generator.
func NewGenerator(cfg *config.Config) *Generator {
	return &Generator{
		cfg:         cfg,
		templateDir: "web/pdf-templates",
	}
}

// GenerateFromTemplate renders an HTML template with data and returns PDF bytes.
func (g *Generator) GenerateFromTemplate(templateName string, data interface{}) ([]byte, error) {
	html, err := g.renderTemplate(templateName, data)
	if err != nil {
		return nil, fmt.Errorf("pdf: failed to render template %s: %w", templateName, err)
	}
	return g.htmlToPDF(html)
}

// renderTemplate renders a Go html/template with the given data.
func (g *Generator) renderTemplate(templateName string, data interface{}) (string, error) {
	templatePath := filepath.Join(g.templateDir, templateName)

	tmpl, err := template.New(templateName).Funcs(templateFuncs()).ParseFiles(templatePath)
	if err != nil {
		return "", fmt.Errorf("pdf: failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("pdf: failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// htmlToPDF converts an HTML string to PDF bytes using headless Chrome via os/exec.
func (g *Generator) htmlToPDF(html string) ([]byte, error) {
	chromiumPath := g.resolveChromiumPath()

	// Write HTML to a temp file
	tmpHTML, err := os.CreateTemp("", "school-pdf-*.html")
	if err != nil {
		return nil, fmt.Errorf("pdf: failed to create temp HTML file: %w", err)
	}
	defer os.Remove(tmpHTML.Name())

	if _, err := tmpHTML.WriteString(html); err != nil {
		return nil, fmt.Errorf("pdf: failed to write HTML: %w", err)
	}
	tmpHTML.Close()

	// Output PDF path
	tmpPDF := tmpHTML.Name() + ".pdf"
	defer os.Remove(tmpPDF)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := []string{
		"--headless",
		"--no-sandbox",
		"--disable-gpu",
		"--disable-dev-shm-usage",
		"--disable-software-rasterizer",
		"--print-to-pdf=" + tmpPDF,
		"--print-to-pdf-no-header",
		"--no-pdf-header-footer",
		"file://" + tmpHTML.Name(),
	}

	cmd := exec.CommandContext(ctx, chromiumPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdf: chrome failed: %w — stderr: %s", err, stderr.String())
	}

	pdfBytes, err := os.ReadFile(tmpPDF)
	if err != nil {
		return nil, fmt.Errorf("pdf: failed to read generated PDF: %w", err)
	}

	log.Debug().Int("bytes", len(pdfBytes)).Msg("pdf: generated successfully")
	return pdfBytes, nil
}

// resolveChromiumPath finds the Chromium/Chrome binary.
func (g *Generator) resolveChromiumPath() string {
	if g.cfg.ChromiumPath != "" {
		if _, err := os.Stat(g.cfg.ChromiumPath); err == nil {
			return g.cfg.ChromiumPath
		}
	}
	for _, path := range []string{
		"/usr/bin/google-chrome",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/local/bin/chromium",
	} {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return "google-chrome"
}

// templateFuncs returns custom template functions available in PDF templates.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("2 January 2006")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("2 January 2006, 3:04 PM")
		},
		"ordinal": func(n int) string {
			switch {
			case n%100 >= 11 && n%100 <= 13:
				return fmt.Sprintf("%dth", n)
			case n%10 == 1:
				return fmt.Sprintf("%dst", n)
			case n%10 == 2:
				return fmt.Sprintf("%dnd", n)
			case n%10 == 3:
				return fmt.Sprintf("%drd", n)
			default:
				return fmt.Sprintf("%dth", n)
			}
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	}
}

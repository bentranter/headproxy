package headproxy

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	uri "net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Map returns an HTML template func map, using r as the given request
// context.
func Map(w http.ResponseWriter, r *http.Request) map[string]interface{} {
	return map[string]interface{}{
		"headproxy": func(url string) template.HTML {
			return ExtractContent(w, r, url)
		},
	}
}

// ExtractContent extracts the <head> content from the given URL.
func ExtractContent(w http.ResponseWriter, r *http.Request, url string) template.HTML {
	abs, body, err := fetch(w, r, url)
	if err != nil {
		return handleErr(err)
	}

	content, err := extract(body)
	if err != nil {
		return handleErr(err)
	}

	replaced, err := replaceRelativePaths(abs, content)
	if err != nil {
		return handleErr(err)
	}

	return template.HTML(replaced)
}

func handleErr(err error) template.HTML {
	msg := err.Error()
	fmt.Printf("[error] %s\n", msg)
	return template.HTML(`<script>window.addEventListener("DOMContentLoaded", function() {
var div = document.createElement("div")
div.style.background = "rgb(220 38 38)"
div.style.color = "rgb(254 226 226)"
div.style.padding = "1rem"
div.style.margin = "1rem"
div.style.borderRadius = "8px"
div.style.fontFamily = "system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif"
div.style.fontSize = "1rem"
var span = document.createElement("span")
span.style.fontSize = ".875rem"
span.style.textTransform = "uppercase"
span.style.fontWeight = "700"
span.style.marginRight = ".5rem"
span.append("Error:")
div.append(span)
div.append("` + template.JSEscapeString(msg) + `")
document.body.prepend(div)
})
</script>
`)
}

// fetch gets and returns the body of the given URL. If no scheme is given, an
// HTTPS scheme is assumed. The generated absolute URL is the first return.
func fetch(w http.ResponseWriter, r *http.Request, url string) (string, []byte, error) {
	parsed, err := uri.Parse(url)
	if err != nil {
		return "", nil, fmt.Errorf("headproxy: failed to parse URL %s: %w", url, err)
	}
	if parsed.Scheme == "" {
		parsed.Scheme = "https"
	}
	absoluteURL := parsed.String()

	// TODO Consider if more of the request context (like the path, body, or
	// method) should be included here.
	req, err := http.NewRequest(http.MethodGet, absoluteURL, nil)
	if err != nil {
		return "", nil, fmt.Errorf("headproxy: failed to create http request to URL %s: %w", parsed, err)
	}

	// Replace the request header.
	req.Header = r.Header

	resp, err := http.Get(absoluteURL)
	if err != nil {
		return "", nil, fmt.Errorf("headproxy: failed to GET URL %s: %w", parsed, err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("headproxy: failed to read body from URL %s: %w", parsed, err)
	}

	// Replace the outgoing response headers.
	for k, v := range resp.Header {
		w.Header().Set(k, strings.Join(v, ","))
	}

	return absoluteURL, body, resp.Body.Close()
}

// extract extracts the contents of the <head> tag from the given HTML body.
// If the <head> tags aren't present, an error is returned.
func extract(body []byte) ([]byte, error) {
	start := bytes.Index(body, []byte("<head>"))
	if start == -1 {
		return nil, errors.New("headproxy: opening <head> tag not present in payload")
	}

	end := bytes.Index(body, []byte("</head>"))
	if end == -1 {
		return nil, errors.New("headproxy: closing </head> tag not present in payload")
	}

	// The <head> tag is six characters long, so we add that to the index
	// position so that it isn't included in the returned HTML.
	return body[start+6 : end], nil
}

// replaceRelativePaths replaces relative paths in the contents of the <head>
// with absolute paths relative to the given URL.
func replaceRelativePaths(url string, head []byte) (string, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(head))
	if err != nil {
		return "", fmt.Errorf("headproxy: failed to create new document from <head> contents from URL %s: %w", url, err)
	}

	parsedBase, err := uri.Parse(url)
	if err != nil {
		return "", fmt.Errorf("headproxy: failed to parse URL %s: %w", url, err)
	}

	doc.Find("link").Each(func(_ int, s *goquery.Selection) {
		// For each item found, get the href.
		if href, ok := s.Attr("href"); ok {
			parsedRefrence, err := uri.Parse(href)
			if err != nil {
				// Ignore invalid URLs.
				return
			}

			absoluteURL := parsedBase.ResolveReference(parsedRefrence)
			s.SetAttr("href", absoluteURL.String())
		}
	})

	return doc.Find("head").Html()
}

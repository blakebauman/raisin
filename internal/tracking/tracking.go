package tracking

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var hrefRe = regexp.MustCompile(`(?i)href\s*=\s*["']([^"']+)["']`)

// Inject rewrites links for click tracking and adds an open-tracking pixel.
func Inject(html string, emailID uuid.UUID, baseURL string, open, click bool) string {
	if html == "" {
		return html
	}
	base := strings.TrimRight(baseURL, "/")
	out := html
	if click {
		out = hrefRe.ReplaceAllStringFunc(out, func(m string) string {
			sub := hrefRe.FindStringSubmatch(m)
			if len(sub) < 2 {
				return m
			}
			orig := sub[1]
			if strings.HasPrefix(orig, "mailto:") || strings.HasPrefix(orig, "#") || strings.HasPrefix(orig, "{{") {
				return m
			}
			if strings.Contains(strings.ToLower(orig), "/unsubscribe/") {
				return m
			}
			tracked := fmt.Sprintf("%s/t/c/%s?u=%s", base, emailID.String(), url.QueryEscape(orig))
			return strings.Replace(m, orig, tracked, 1)
		})
	}
	if open {
		pixel := fmt.Sprintf(`<img src="%s/t/o/%s" width="1" height="1" alt="" style="display:none" />`, base, emailID.String())
		if strings.Contains(strings.ToLower(out), "</body>") {
			idx := strings.LastIndex(strings.ToLower(out), "</body>")
			out = out[:idx] + pixel + out[idx:]
		} else {
			out += pixel
		}
	}
	return out
}

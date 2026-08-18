package tracking

import (
	"html"
	"strings"
)

// AppendUnsubscribe adds a visible unsubscribe link to HTML and/or text bodies.
// Call after Inject so click tracking does not rewrite the unsubscribe href.
func AppendUnsubscribe(htmlBody, textBody, unsubURL string) (string, string) {
	unsubURL = strings.TrimSpace(unsubURL)
	if unsubURL == "" {
		return htmlBody, textBody
	}
	if htmlBody != "" && !strings.Contains(strings.ToLower(htmlBody), "/unsubscribe/") {
		footer := `<p style="margin-top:2em;font-size:12px;line-height:1.4;color:#666666"><a href="` +
			html.EscapeString(unsubURL) + `" style="color:#666666">Unsubscribe</a></p>`
		lower := strings.ToLower(htmlBody)
		if idx := strings.LastIndex(lower, "</body>"); idx >= 0 {
			htmlBody = htmlBody[:idx] + footer + htmlBody[idx:]
		} else {
			htmlBody += footer
		}
	}
	if textBody != "" && !strings.Contains(textBody, "/unsubscribe/") {
		textBody = strings.TrimRight(textBody, "\n") + "\n\nUnsubscribe: " + unsubURL + "\n"
	} else if textBody == "" && htmlBody != "" {
		// Ensure multipart text alternative mentions unsubscribe when HTML-only.
		textBody = "Unsubscribe: " + unsubURL + "\n"
	}
	return htmlBody, textBody
}

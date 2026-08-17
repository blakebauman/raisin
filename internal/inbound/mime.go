package inbound

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
)

type ParsedAttachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

// parseMIMEMessage extracts html/text bodies and file attachments from RFC822 bytes.
func parseMIMEMessage(raw []byte) (html, text string, atts []ParsedAttachment) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		h, t := parseMIMEBodies(string(raw))
		return h, t, nil
	}
	html, text, atts = walkPart(msg.Header, msg.Body)
	if html == "" && text == "" && len(atts) == 0 {
		h, t := parseMIMEBodies(string(raw))
		return h, t, nil
	}
	return html, text, atts
}

func walkPart(hdr mail.Header, body io.Reader) (html, text string, atts []ParsedAttachment) {
	contentType := hdr.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = "text/plain"
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			h, t, a := walkPart(mail.Header(p.Header), p)
			if h != "" && html == "" {
				html = h
			}
			if t != "" && text == "" {
				text = t
			}
			atts = append(atts, a...)
		}
		return html, text, atts
	}

	data, err := io.ReadAll(io.LimitReader(body, 25<<20))
	if err != nil {
		return "", "", nil
	}
	data = decodeTransfer(data, strings.ToLower(hdr.Get("Content-Transfer-Encoding")))

	filename := ""
	if disp := hdr.Get("Content-Disposition"); disp != "" {
		if _, dparams, err := mime.ParseMediaType(disp); err == nil {
			filename = dparams["filename"]
		}
		if strings.Contains(strings.ToLower(disp), "attachment") && filename == "" {
			filename = "attachment"
		}
	}
	if filename == "" {
		filename = params["name"]
	}

	dispLower := strings.ToLower(hdr.Get("Content-Disposition"))
	isAttach := strings.Contains(dispLower, "attachment") ||
		(filename != "" && !strings.HasPrefix(mediaType, "text/") && mediaType != "message/rfc822")

	if isAttach {
		if mediaType == "" {
			mediaType = "application/octet-stream"
		}
		if filename == "" {
			filename = "attachment"
		}
		return "", "", []ParsedAttachment{{Filename: filename, ContentType: mediaType, Content: data}}
	}

	switch {
	case strings.HasPrefix(mediaType, "text/html"):
		return string(data), "", nil
	case strings.HasPrefix(mediaType, "text/plain"):
		return "", string(data), nil
	case filename != "":
		return "", "", []ParsedAttachment{{Filename: filename, ContentType: mediaType, Content: data}}
	default:
		return "", string(data), nil
	}
}

func decodeTransfer(data []byte, encoding string) []byte {
	switch encoding {
	case "base64":
		trimmed := make([]byte, 0, len(data))
		for _, b := range data {
			if b != '\n' && b != '\r' && b != ' ' && b != '\t' {
				trimmed = append(trimmed, b)
			}
		}
		out := make([]byte, base64.StdEncoding.DecodedLen(len(trimmed)))
		n, err := base64.StdEncoding.Decode(out, trimmed)
		if err != nil {
			return data
		}
		return out[:n]
	default:
		return data
	}
}

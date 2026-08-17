package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.raisin.run"
	userAgent      = "raisin-cli/0.1.0"
)

type client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	apiKey := os.Getenv("RAISIN_API_KEY")
	baseURL := envOrDefault("RAISIN_BASE_URL", defaultBaseURL)

	global := flag.NewFlagSet("raisin", flag.ContinueOnError)
	global.SetOutput(io.Discard)
	global.StringVar(&apiKey, "api-key", apiKey, "Raisin API key")
	global.StringVar(&baseURL, "base-url", baseURL, "Raisin API base URL")
	if err := global.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Println(usage())
			return nil
		}
		return err
	}
	args = global.Args()

	if len(args) == 0 {
		return errors.New(usage())
	}
	if args[0] == "help" {
		fmt.Println(usage())
		return nil
	}
	if apiKey == "" {
		return errors.New("API key is required; use --api-key or RAISIN_API_KEY")
	}

	c := &client{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 30 * time.Second},
	}

	switch args[0] {
	case "emails":
		return runEmails(c, args[1:])
	case "domains":
		return runDomains(c, args[1:])
	case "contacts":
		return runList(c, args[1:], "/contacts", "contacts")
	case "templates":
		return runList(c, args[1:], "/templates", "templates")
	case "automations":
		return runList(c, args[1:], "/automations", "automations")
	case "ips":
		return runList(c, args[1:], "/ip-pools", "ips")
	default:
		return fmt.Errorf("unknown command %q\n%s", args[0], usage())
	}
}

func runEmails(c *client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: raisin emails <send|list|get>")
	}
	switch args[0] {
	case "send":
		fs := flag.NewFlagSet("emails send", flag.ContinueOnError)
		from := fs.String("from", "", "sender address")
		to := fs.String("to", "", "recipient address")
		subject := fs.String("subject", "", "email subject")
		html := fs.String("html", "", "HTML body")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return errors.New("usage: raisin emails send --from X --to Y --subject Z --html H")
		}
		if *from == "" || *to == "" || *subject == "" || *html == "" {
			return errors.New("--from, --to, --subject, and --html are required")
		}
		return c.request(http.MethodPost, "/emails", map[string]string{
			"from": *from, "to": *to, "subject": *subject, "html": *html,
		})
	case "list":
		return requireNoArgsAndRequest(c, args[1:], http.MethodGet, "/emails")
	case "get":
		if len(args) != 2 {
			return errors.New("usage: raisin emails get <id>")
		}
		return c.request(http.MethodGet, "/emails/"+args[1], nil)
	default:
		return fmt.Errorf("unknown emails command %q", args[0])
	}
}

func runDomains(c *client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: raisin domains <list|create>")
	}
	switch args[0] {
	case "list":
		return requireNoArgsAndRequest(c, args[1:], http.MethodGet, "/domains")
	case "create":
		fs := flag.NewFlagSet("domains create", flag.ContinueOnError)
		name := fs.String("name", "", "domain name")
		region := fs.String("region", "us-east-1", "sending region")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 0 || *name == "" {
			return errors.New("usage: raisin domains create --name example.com [--region us-east-1]")
		}
		return c.request(http.MethodPost, "/domains", map[string]string{
			"name": *name, "region": *region,
		})
	default:
		return fmt.Errorf("unknown domains command %q", args[0])
	}
}

func runList(c *client, args []string, path, resource string) error {
	if len(args) != 1 || args[0] != "list" {
		return fmt.Errorf("usage: raisin %s list", resource)
	}
	return c.request(http.MethodGet, path, nil)
}

func requireNoArgsAndRequest(c *client, args []string, method, path string) error {
	if len(args) != 0 {
		return errors.New("unexpected arguments")
	}
	return c.request(method, path, nil)
}

func (c *client) request(method, path string, body any) error {
	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, payload)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	printJSON(data)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API returned %s", resp.Status)
	}
	return nil
}

func printJSON(data []byte) {
	var value any
	if len(data) > 0 && json.Unmarshal(data, &value) == nil {
		pretty, _ := json.MarshalIndent(value, "", "  ")
		fmt.Println(string(pretty))
		return
	}
	fmt.Println(string(data))
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func usage() string {
	return `usage: raisin [--api-key KEY] [--base-url URL] <command>

commands:
  emails send --from X --to Y --subject Z --html H
  emails list
  emails get <id>
  domains list
  domains create --name example.com [--region us-east-1]
  contacts list
  templates list
  automations list
  ips list`
}

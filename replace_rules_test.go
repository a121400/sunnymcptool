package main

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func useRules(t *testing.T, items ...compiledRule) {
	t.Helper()
	storeReplaceRules(items)
	t.Cleanup(func() { storeReplaceRules(nil) })
}

func TestReplaceRequestHeaderSetAndScope(t *testing.T) {
	rule, err := compileReplaceRule("请求头", "User-Agent", "Android", "example.com")
	if err != nil {
		t.Fatal(err)
	}
	useRules(t, rule)
	h := http.Header{"User-Agent": []string{"Windows"}}
	ReplaceRequestHeader(h, "https://other.test/")
	if h.Get("User-Agent") != "Windows" {
		t.Fatalf("scope should skip, got %q", h.Get("User-Agent"))
	}
	ReplaceRequestHeader(h, "https://example.com/a")
	if h.Get("User-Agent") != "Android" {
		t.Fatalf("header not replaced, got %q", h.Get("User-Agent"))
	}

	del, err := compileReplaceRule("请求头", "User-Agent", "", "")
	if err != nil {
		t.Fatal(err)
	}
	useRules(t, del)
	ReplaceRequestHeader(h, "https://example.com/a")
	if h.Get("User-Agent") != "" {
		t.Fatalf("header not deleted, got %q", h.Get("User-Agent"))
	}
}

func TestReplaceHeaderFragmentAndBodyRegex(t *testing.T) {
	headerRule, err := compileReplaceRule("请求头", "User-Agent||Windows", "Linux", "")
	if err != nil {
		t.Fatal(err)
	}
	bodyRule, err := compileReplaceRule("正则响应", `id=(\d+)`, "id=ok-$1", "re:api\\.test")
	if err != nil {
		t.Fatal(err)
	}
	urlRule, err := compileReplaceRule("仅URL", "http://", "https://", "")
	if err != nil {
		t.Fatal(err)
	}
	useRules(t, headerRule, bodyRule, urlRule)

	h := http.Header{"user-agent": []string{"Windows NT"}}
	ReplaceRequestHeader(h, "https://api.test/x")
	if got := h["user-agent"][0]; got != "Linux NT" {
		t.Fatalf("fragment replace got %q", got)
	}

	body := ReplaceHTTPBody([]byte("id=7"), "https://api.test/x", true)
	if string(body) != "id=ok-7" {
		t.Fatalf("regex body got %q", body)
	}
	skipped := ReplaceHTTPBody([]byte("id=7"), "https://other.test/x", true)
	if string(skipped) != "id=7" {
		t.Fatalf("regex scope failed, got %q", skipped)
	}

	u, _ := url.Parse("http://api.test/x")
	nu, file := ReplaceURL(u)
	if file != nil || nu.String() != "https://api.test/x" {
		t.Fatalf("url replace got %s file %q", nu, file)
	}
	same := ReplaceHTTPBody([]byte("http://keep"), "https://api.test/x", false)
	if string(same) != "http://keep" {
		t.Fatalf("仅URL changed body: %q", same)
	}
}

func TestResponseFileReread(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "login.js")
	if err := os.WriteFile(file, []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}
	rule, err := compileReplaceRule("响应文件", "login.js", file, "")
	if err != nil {
		t.Fatal(err)
	}
	useRules(t, rule)
	if err := os.WriteFile(file, []byte("v2"), 0644); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse("https://cdn.test/login.js")
	_, body := ReplaceURL(u)
	if string(body) != "v2" {
		t.Fatalf("file not reread, got %q", body)
	}
	if ct := ContentTypeByURL(u); ct != "application/javascript; charset=utf-8" {
		t.Fatalf("content type %q", ct)
	}
}

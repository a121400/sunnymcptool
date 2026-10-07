package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/qtgolang/SunnyNet/src/encoding/hex"
	"github.com/qtgolang/SunnyNet/src/protobuf/JSON"
)

const (
	ruleBytes = iota + 1
	ruleFile
	ruleReqHeader
	ruleRespHeader
	ruleReqBody
	ruleRespBody
	ruleRegexReq
	ruleRegexResp
	ruleURL
)

const replaceTypeList = "Base64、HEX、String(UTF8)、String(GBK)、响应文件、请求头、响应头、请求体、响应体、正则请求、正则响应、仅URL"

type compiledRule struct {
	kind     int
	source   []byte
	target   []byte
	scope    string
	scopeRe  *regexp.Regexp
	header   string
	oldPart  []byte
	setAll   bool
	re       *regexp.Regexp
	filePath string
}

var (
	_ReplaceRules  []compiledRule
	replaceRulesMu sync.RWMutex
)

func (r *compiledRule) match(rawURL string) bool {
	if r.scopeRe != nil {
		return r.scopeRe.MatchString(rawURL)
	}
	if r.scope == "" {
		return true
	}
	return strings.Contains(rawURL, r.scope)
}

func snapshotReplaceRules() []compiledRule {
	replaceRulesMu.RLock()
	defer replaceRulesMu.RUnlock()
	out := make([]compiledRule, len(_ReplaceRules))
	copy(out, _ReplaceRules)
	return out
}

func storeReplaceRules(rules []compiledRule) {
	replaceRulesMu.Lock()
	_ReplaceRules = rules
	replaceRulesMu.Unlock()
}

func compileReplaceRule(ruleType, source, target, scope string) (compiledRule, error) {
	var rule compiledRule
	scope = strings.TrimSpace(scope)
	rule.scope = scope
	if strings.HasPrefix(scope, "re:") {
		re, err := regexp.Compile(scope[3:])
		if err != nil {
			return rule, errors.New("生效范围正则无效: " + err.Error())
		}
		rule.scopeRe = re
		rule.scope = ""
	}
	if source == "" && ruleType != "请求头" && ruleType != "响应头" {
		return rule, errors.New("源内容不能为空")
	}
	switch ruleType {
	case "Base64":
		bs1, err := base64.StdEncoding.DecodeString(source)
		if err != nil {
			return rule, errors.New("源内容不是合法 Base64")
		}
		bs2, err := base64.StdEncoding.DecodeString(target)
		if err != nil {
			return rule, errors.New("替换内容不是合法 Base64")
		}
		rule.kind = ruleBytes
		rule.source = bs1
		rule.target = bs2
	case "HEX":
		bs1, err := hex.DecodeString(source)
		if err != nil {
			return rule, errors.New("源内容不是合法 HEX")
		}
		bs2, err := hex.DecodeString(target)
		if err != nil {
			return rule, errors.New("替换内容不是合法 HEX")
		}
		rule.kind = ruleBytes
		rule.source = bs1
		rule.target = bs2
	case "String(UTF8)":
		rule.kind = ruleBytes
		rule.source = []byte(source)
		rule.target = []byte(target)
	case "String(GBK)":
		rule.kind = ruleBytes
		rule.source = Utf8ToGBK([]byte(source))
		rule.target = Utf8ToGBK([]byte(target))
	case "响应文件":
		bs, err := os.ReadFile(target)
		if err != nil {
			return rule, errors.New("响应文件不存在或无法读取: " + target)
		}
		rule.kind = ruleFile
		rule.source = []byte(source)
		rule.target = bs
		rule.filePath = target
	case "请求头", "响应头":
		name, old, setAll, err := parseHeaderSpec(source)
		if err != nil {
			return rule, err
		}
		if ruleType == "请求头" {
			rule.kind = ruleReqHeader
		} else {
			rule.kind = ruleRespHeader
		}
		rule.header = name
		rule.oldPart = old
		rule.setAll = setAll
		rule.target = []byte(target)
	case "请求体":
		rule.kind = ruleReqBody
		rule.source = []byte(source)
		rule.target = []byte(target)
	case "响应体":
		rule.kind = ruleRespBody
		rule.source = []byte(source)
		rule.target = []byte(target)
	case "正则请求", "正则响应":
		re, err := regexp.Compile(source)
		if err != nil {
			return rule, errors.New("正则无效: " + err.Error())
		}
		if ruleType == "正则请求" {
			rule.kind = ruleRegexReq
		} else {
			rule.kind = ruleRegexResp
		}
		rule.re = re
		rule.target = []byte(target)
	case "仅URL":
		rule.kind = ruleURL
		rule.source = []byte(source)
		rule.target = []byte(target)
	default:
		return rule, errors.New("无效的替换类型: " + ruleType + "，支持: " + replaceTypeList)
	}
	return rule, nil
}

func parseHeaderSpec(src string) (name string, old []byte, setAll bool, err error) {
	src = strings.TrimSpace(src)
	if src == "" {
		return "", nil, false, errors.New("源内容应填写头名，或 头名||要替换的片段")
	}
	if i := strings.Index(src, "||"); i >= 0 {
		name = strings.TrimSpace(src[:i])
		old = []byte(src[i+2:])
		if name == "" || len(old) == 0 {
			return "", nil, false, errors.New("头名||片段 两边都不能为空")
		}
		return name, old, false, nil
	}
	return src, nil, true, nil
}

func RebuildReplaceRulesFromConfig() {
	var rules []compiledRule
	for _, v := range GlobalConfig.ReplaceRules {
		rule, err := compileReplaceRule(v.Type, v.Src, v.Dest, v.Scope)
		if err != nil {
			continue
		}
		rules = append(rules, rule)
	}
	storeReplaceRules(rules)
}

func ReplaceRulesEvent(command string, args *JSON.SyJson) any {
	switch command {
	case "保存替换规则":
		_TmpLock.Lock()
		defer _TmpLock.Unlock()
		var failHash []string
		var rules []compiledRule
		var saved []ConfigReplaceRules
		for i := 0; i < args.GetNum("Data"); i++ {
			prefix := "Data[" + strconv.Itoa(i) + "]."
			hash := args.GetData(prefix + "Hash")
			ruleType := args.GetData(prefix + "替换类型")
			source := unescapeRuleText(args.GetData(prefix + "源内容"))
			target := unescapeRuleText(args.GetData(prefix + "替换内容"))
			scope := unescapeRuleText(args.GetData(prefix + "生效范围"))
			rule, err := compileReplaceRule(ruleType, source, target, scope)
			if err != nil {
				failHash = append(failHash, hash)
				continue
			}
			rules = append(rules, rule)
			saved = append(saved, ConfigReplaceRules{Type: ruleType, Hash: hash, Src: source, Dest: target, Scope: scope})
		}
		GlobalConfig.ReplaceRules = saved
		_ = GlobalConfig.saveToFile()
		storeReplaceRules(rules)
		return failHash
	default:
		return HostsRulesEvent(command, args)
	}
}

func unescapeRuleText(s string) string {
	s = strings.ReplaceAll(s, "\\\\", "\\")
	s = strings.ReplaceAll(s, "\\\"", "\"")
	return s
}

func ReplaceURL(u *url.URL) (*url.URL, []byte) {
	if u == nil {
		return u, nil
	}
	raw := u.String()
	rules := snapshotReplaceRules()
	ur := raw
	var filePath string
	var cached []byte
	matched := false
	for i := range rules {
		r := &rules[i]
		if !r.match(raw) {
			continue
		}
		switch r.kind {
		case ruleBytes, ruleURL:
			if len(r.source) > 0 && bytes.Contains([]byte(ur), r.source) {
				ur = string(bytes.ReplaceAll([]byte(ur), r.source, r.target))
				matched = true
			}
		case ruleFile:
			if len(r.source) > 0 && bytes.Contains([]byte(ur), r.source) {
				filePath = r.filePath
				cached = append([]byte(nil), r.target...)
				matched = true
				goto fileDone
			}
		}
	}
fileDone:
	if !matched {
		return u, nil
	}
	if filePath != "" {
		if b, err := os.ReadFile(filePath); err == nil {
			cached = b
		}
	}
	um, err := url.Parse(ur)
	if err != nil {
		if filePath != "" {
			return u, cached
		}
		return u, nil
	}
	if filePath != "" {
		return um, cached
	}
	return um, nil
}

func ContentTypeByURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	switch strings.ToLower(path.Ext(u.Path)) {
	case ".js", ".mjs":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".wasm":
		return "application/wasm"
	case ".txt":
		return "text/plain; charset=utf-8"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	default:
		return ""
	}
}

func ReplaceRequestHeader(header http.Header, rawURL string) {
	replaceHeader(header, rawURL, false)
}

func ReplaceResponseHeader(header http.Header, rawURL string) {
	replaceHeader(header, rawURL, true)
}

func replaceHeader(header http.Header, rawURL string, response bool) {
	if header == nil {
		return
	}
	rules := snapshotReplaceRules()
	for key, values := range header {
		for i, value := range values {
			header[key][i] = string(applyByteRules([]byte(value), rawURL, rules))
		}
	}
	applyNamedHeaders(header, rawURL, response, rules)
}

func ReplaceHTTPBody(b []byte, rawURL string, response bool) []byte {
	rules := snapshotReplaceRules()
	b = applyByteRules(b, rawURL, rules)
	for i := range rules {
		r := &rules[i]
		if !r.match(rawURL) {
			continue
		}
		switch r.kind {
		case ruleReqBody:
			if !response {
				b = replaceNeedle(b, r.source, r.target)
			}
		case ruleRespBody:
			if response {
				b = replaceNeedle(b, r.source, r.target)
			}
		case ruleRegexReq:
			if !response && r.re != nil {
				b = r.re.ReplaceAll(b, r.target)
			}
		case ruleRegexResp:
			if response && r.re != nil {
				b = r.re.ReplaceAll(b, r.target)
			}
		}
	}
	return b
}

func ReplaceBody(b []byte) []byte {
	return applyByteRules(b, "", snapshotReplaceRules())
}

func applyByteRules(b []byte, rawURL string, rules []compiledRule) []byte {
	for i := range rules {
		r := &rules[i]
		if r.kind != ruleBytes || !r.match(rawURL) {
			continue
		}
		b = replaceNeedle(b, r.source, r.target)
	}
	return b
}

func replaceNeedle(b, old, repl []byte) []byte {
	if len(old) == 0 || !bytes.Contains(b, old) {
		return b
	}
	return bytes.ReplaceAll(b, old, repl)
}

func applyNamedHeaders(header http.Header, rawURL string, response bool, rules []compiledRule) {
	for i := range rules {
		r := &rules[i]
		if !r.match(rawURL) || r.header == "" {
			continue
		}
		if response && r.kind != ruleRespHeader {
			continue
		}
		if !response && r.kind != ruleReqHeader {
			continue
		}
		if r.setAll {
			setHeaderValue(header, r.header, string(r.target))
			continue
		}
		replaceHeaderPart(header, r.header, r.oldPart, r.target)
	}
}

func setHeaderValue(header http.Header, name, value string) {
	canon := http.CanonicalHeaderKey(name)
	found := false
	for key := range header {
		if http.CanonicalHeaderKey(key) != canon {
			continue
		}
		if !found && value != "" {
			header[key] = []string{value}
			found = true
			continue
		}
		delete(header, key)
	}
	if !found && value != "" {
		header[canon] = []string{value}
	}
}

func replaceHeaderPart(header http.Header, name string, old, repl []byte) {
	canon := http.CanonicalHeaderKey(name)
	for key, values := range header {
		if http.CanonicalHeaderKey(key) != canon {
			continue
		}
		for i, value := range values {
			header[key][i] = string(replaceNeedle([]byte(value), old, repl))
		}
	}
}

func UpdateContentLength(header http.Header, oldLen, newLen int) {
	if header == nil || oldLen == newLen {
		return
	}
	for key := range header {
		if strings.EqualFold(key, "Content-Length") {
			delete(header, key)
		}
	}
	header["Content-Length"] = []string{strconv.Itoa(newLen)}
}

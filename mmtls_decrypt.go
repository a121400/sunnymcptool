package main

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// 手机短连接加密用的 psk_access。换成你自己生成的 32 字节，
// 手机必须用同一把 psk 加密，抓到的 /mmtls/ 正文才能解开。
// 文件在 %USERPROFILE%\Sunny\mmtls_key.txt，每次抓包重新读取。
//
// psk=64位十六进制
// 或者直接填已经展开的流量密钥（16 字节 key + 12 字节 iv）：
// client_key= client_iv= server_key= server_iv=

type mmtlsKeys struct {
	psk        []byte
	clientKey  []byte
	clientIV   []byte
	serverKey  []byte
	serverIV   []byte
}

type mmtlsSess struct {
	newsend   []byte
	psk       []byte
	serverKey []byte
	serverIV  []byte
}

var (
	mmtlsMu    sync.Mutex
	mmtlsCache mmtlsKeys
	mmtlsStamp int64
	mmtlsSessMu sync.Mutex
	mmtlsByID  = map[int]mmtlsSess{}
)

func mmtlsKeyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "mmtls_key.txt"
	}
	dir := filepath.Join(home, "Sunny")
	_ = os.MkdirAll(dir, 0777)
	return filepath.Join(dir, "mmtls_key.txt")
}

func ensureMMTLSKeyFile() {
	path := mmtlsKeyPath()
	if _, err := os.Stat(path); err == nil {
		return
	}
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	text := "# 手机 mmtls 短连接的 psk_access，必须和这里同一把。\r\n# 改这个文件后不用重启 Sunny，下一条 /mmtls/ 会按新密钥解。\r\npsk=" + hex.EncodeToString(buf) + "\r\n"
	_ = os.WriteFile(path, []byte(text), 0644)
}

func loadMMTLSKeys() mmtlsKeys {
	ensureMMTLSKeyFile()
	path := mmtlsKeyPath()
	st, err := os.Stat(path)
	if err != nil {
		return mmtlsKeys{}
	}
	mmtlsMu.Lock()
	defer mmtlsMu.Unlock()
	if st.ModTime().UnixNano() == mmtlsStamp {
		return mmtlsCache
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return mmtlsKeys{}
	}
	var k mmtlsKeys
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.IndexByte(line, '=')
		if i < 0 {
			continue
		}
		name := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		b, err := hex.DecodeString(val)
		if err != nil {
			continue
		}
		switch name {
		case "psk":
			k.psk = b
		case "client_key":
			k.clientKey = b
		case "client_iv":
			k.clientIV = b
		case "server_key":
			k.serverKey = b
		case "server_iv":
			k.serverIV = b
		}
	}
	mmtlsCache = k
	mmtlsStamp = st.ModTime().UnixNano()
	return k
}

func isMMTLS(url string, body []byte) bool {
	if strings.Contains(strings.ToLower(url), "/mmtls/") {
		return true
	}
	return len(body) >= 5 && (body[0] == 0x16 || body[0] == 0x19 || body[0] == 0x17) && body[1] == 0xF1
}

type mmtlsRec struct {
	typ  byte
	hdr  []byte
	body []byte
}

func splitMMTLS(data []byte) []mmtlsRec {
	var out []mmtlsRec
	i := 0
	for i+5 <= len(data) {
		ln := int(binary.BigEndian.Uint16(data[i+3 : i+5]))
		if ln < 0 || i+5+ln > len(data) {
			break
		}
		hdr := append([]byte(nil), data[i:i+5]...)
		body := append([]byte(nil), data[i+5:i+5+ln]...)
		out = append(out, mmtlsRec{typ: data[i], hdr: hdr, body: body})
		i += 5 + ln
	}
	return out
}

func hkdfExpandSHA256(prk, info []byte, n int) []byte {
	out := make([]byte, 0, n)
	var t []byte
	for i := byte(1); len(out) < n; i++ {
		mac := hmac.New(sha256.New, prk)
		mac.Write(t)
		mac.Write(info)
		mac.Write([]byte{i})
		t = mac.Sum(nil)
		out = append(out, t...)
	}
	return out[:n]
}

func ivXor(iv []byte, seq int) []byte {
	nonce := append([]byte(nil), iv...)
	last := binary.BigEndian.Uint32(iv[8:12]) ^ uint32(seq)
	binary.BigEndian.PutUint32(nonce[8:12], last)
	return nonce
}

func gcmOpen(key, nonce, ct, aad []byte) ([]byte, bool) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, false
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, false
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, false
	}
	pt, err := gcm.Open(nil, nonce, ct, aad)
	if err != nil {
		return nil, false
	}
	return pt, true
}

func gcmSeal(key, nonce, pt, aad []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil
	}
	return gcm.Seal(nil, nonce, pt, aad)
}

func aadOf(seq int, hdr []byte) []byte {
	a := make([]byte, 8+len(hdr))
	binary.BigEndian.PutUint64(a[:8], uint64(seq))
	copy(a[8:], hdr)
	return a
}

func decryptRecords(recs []mmtlsRec, key, iv []byte, seq0 int, want byte) []byte {
	var out []byte
	seq := seq0
	for _, rec := range recs {
		if want != 0 && rec.typ != want && rec.typ != 0x15 {
			seq++
			continue
		}
		pt, ok := gcmOpen(key, ivXor(iv, seq), rec.body, aadOf(seq, rec.hdr))
		seq++
		if ok && rec.typ == want {
			out = append(out, pt...)
		}
	}
	return out
}

// MMTLSDecryptRequest 用当前密钥文件解开短连接请求里的 0x17 业务记录。
// 只改展示用的正文，不改发出去的密文。
func MMTLSDecryptRequest(id int, url string, body []byte) ([]byte, bool) {
	if !isMMTLS(url, body) {
		return nil, false
	}
	keys := loadMMTLSKeys()
	recs := splitMMTLS(body)
	if len(recs) == 0 {
		return nil, false
	}
	if len(keys.clientKey) >= 16 && len(keys.clientIV) == 12 {
		enc := recs
		seq0 := 1
		if recs[0].typ == 0x19 || recs[0].typ == 0x16 {
			enc = recs[1:]
		} else {
			seq0 = 0
		}
		pt := decryptRecords(enc, keys.clientKey[:16], keys.clientIV, seq0, 0x17)
		if len(pt) > 0 {
			mmtlsSessMu.Lock()
			mmtlsByID[id] = mmtlsSess{serverKey: keys.serverKey, serverIV: keys.serverIV}
			mmtlsSessMu.Unlock()
			return presentMMTLS(pt), true
		}
	}
	if len(keys.psk) < 16 || (recs[0].typ != 0x19 && recs[0].typ != 0x16) {
		return nil, false
	}
	tried := map[string]struct{}{}
	var list [][]byte
	list = append(list, keys.psk)
	list = append(list, loadPSKCandidates()...)
	for _, psk := range list {
		if len(psk) < 16 {
			continue
		}
		idhex := hex.EncodeToString(psk)
		if _, ok := tried[idhex]; ok {
			continue
		}
		tried[idhex] = struct{}{}
		pt, ok := decryptRequestPSK(id, recs, psk)
		if !ok {
			continue
		}
		if !bytes.Equal(psk, keys.psk) {
			pinPSK(psk)
		}
		return pt, true
	}
	return nil, false
}

func decryptRequestPSK(id int, recs []mmtlsRec, psk []byte) ([]byte, bool) {
	if len(psk) < 16 || len(recs) == 0 || (recs[0].typ != 0x19 && recs[0].typ != 0x16) {
		return nil, false
	}
	packed := recs[0].body
	sum := sha256.Sum256(packed)
	short := hkdfExpandSHA256(psk, append([]byte("early data key expansion"), sum[:]...), 28)
	key, iv := short[:16], short[16:28]
	var part2 []byte
	var app []byte
	seq := 1
	for _, rec := range recs[1:] {
		pt, ok := gcmOpen(key, ivXor(iv, seq), rec.body, aadOf(seq, rec.hdr))
		seq++
		if !ok {
			continue
		}
		if rec.typ == 0x19 && part2 == nil {
			part2 = pt
		}
		if rec.typ == 0x17 {
			app = append(app, pt...)
		}
	}
	if len(app) == 0 {
		return nil, false
	}
	news := append(append([]byte{}, packed...), part2...)
	mmtlsSessMu.Lock()
	mmtlsByID[id] = mmtlsSess{newsend: news, psk: append([]byte{}, psk...)}
	mmtlsSessMu.Unlock()
	return presentMMTLS(app), true
}

func candidatePath() string {
	return filepath.Join(filepath.Dir(mmtlsKeyPath()), "mmtls_candidates.txt")
}

func loadPSKCandidates() [][]byte {
	raw, err := os.ReadFile(candidatePath())
	if err != nil {
		return nil
	}
	var out [][]byte
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if len(line) != 64 {
			continue
		}
		b, err := hex.DecodeString(line)
		if err != nil || len(b) != 32 {
			continue
		}
		out = append(out, b)
		if len(out) >= 40 {
			break
		}
	}
	return out
}

func pinPSK(psk []byte) {
	if len(psk) != 32 {
		return
	}
	path := mmtlsKeyPath()
	raw, _ := os.ReadFile(path)
	enc := hex.EncodeToString(psk)
	var out []string
	found := false
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "psk=") {
			out = append(out, "psk="+enc)
			found = true
			continue
		}
		out = append(out, strings.TrimRight(line, "\r"))
	}
	if !found {
		out = append(out, "psk="+enc)
	}
	_ = os.WriteFile(path, []byte(strings.Join(out, "\n")), 0644)
}

// presentMMTLS 把短连接明文改成从路径开头的文本。
// 原始字节以 00 00 开头，文本视图会把接口路径藏起来。
func presentMMTLS(plain []byte) []byte {
	url, host, body, ok := splitBiz(plain)
	if !ok {
		return plain
	}
	head := url + "\r\nHost: " + host + "\r\n\r\n"
	if text := matchBizText(url); len(text) > 0 {
		return append([]byte(head), text...)
	}
	if mostlyText(body) {
		return append([]byte(head), body...)
	}
	return []byte(fmt.Sprintf("%s业务包 %d 字节。mmtls 已解开，里面没有可读文本。", head, len(body)))
}

func freshBizText() []byte {
	return freshFileText("mmtls_biz.bin")
}

func freshFileText(name string) []byte {
	path := filepath.Join(filepath.Dir(mmtlsKeyPath()), name)
	st, err := os.Stat(path)
	if err != nil || time.Since(st.ModTime()) > 8*time.Second {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) < 16 {
		return nil
	}
	return stringsFrom(raw)
}

func stringsFrom(raw []byte) []byte {
	const maxLine = 48 * 1024
	const maxTotal = 64 * 1024
	var lines []string
	var cur []byte
	total := 0
	flush := func() {
		if len(cur) >= 6 && total < maxTotal {
			line := string(cur)
			if len(line) > maxLine {
				line = line[:maxLine]
			}
			if len(lines) == 0 || lines[len(lines)-1] != line {
				lines = append(lines, line)
				total += len(line) + 2
			}
		}
		cur = cur[:0]
	}
	for _, c := range raw {
		if c >= 32 && c < 127 {
			cur = append(cur, c)
			continue
		}
		flush()
		if total >= maxTotal {
			break
		}
	}
	flush()
	if len(lines) == 0 {
		return nil
	}
	return []byte(strings.Join(lines, "\r\n"))
}

func splitBiz(plain []byte) (url, host string, body []byte, ok bool) {
	if len(plain) < 8 {
		return "", "", nil, false
	}
	i := 4
	ulen := int(binary.BigEndian.Uint16(plain[i : i+2]))
	i += 2
	if ulen < 1 || ulen > 512 || i+ulen+2 > len(plain) {
		return "", "", nil, false
	}
	rawURL := plain[i : i+ulen]
	if len(rawURL) == 0 || rawURL[0] != '/' {
		return "", "", nil, false
	}
	i += ulen
	hlen := int(binary.BigEndian.Uint16(plain[i : i+2]))
	i += 2
	if hlen < 1 || hlen > 256 || i+hlen > len(plain) {
		return "", "", nil, false
	}
	rawHost := plain[i : i+hlen]
	i += hlen
	if i+4 <= len(plain) {
		blen := int(binary.BigEndian.Uint32(plain[i : i+4]))
		i += 4
		if blen >= 0 && i+blen <= len(plain) {
			return string(rawURL), string(rawHost), plain[i : i+blen], true
		}
	}
	return string(rawURL), string(rawHost), plain[i:], true
}

// MMTLSDecryptResponse 用同一次请求留下的握手材料解开响应。
func MMTLSDecryptResponse(id int, url string, body []byte) ([]byte, bool) {
	if !isMMTLS(url, body) {
		return nil, false
	}
	mmtlsSessMu.Lock()
	sess, ok := mmtlsByID[id]
	mmtlsSessMu.Unlock()
	if !ok {
		return nil, false
	}
	recs := splitMMTLS(body)
	if len(recs) == 0 {
		return nil, false
	}
	if len(sess.serverKey) >= 16 && len(sess.serverIV) == 12 {
		pt := decryptRecords(recs, sess.serverKey[:16], sess.serverIV, 0, 0x17)
		if len(pt) == 0 {
			pt = decryptRecords(recs, sess.serverKey[:16], sess.serverIV, 1, 0x17)
		}
		if len(pt) > 0 {
			return finishResponse(pt), true
		}
	}
	if len(sess.psk) < 16 || len(sess.newsend) == 0 {
		return nil, false
	}
	decKey := []byte(nil)
	decIV := []byte(nil)
	var app []byte
	for i, rec := range recs {
		if i == 0 && rec.typ == 0x16 {
			h := sha256.Sum256(append(append([]byte{}, sess.newsend...), rec.body...))
			short := hkdfExpandSHA256(sess.psk, append([]byte("handshake key expansion"), h[:]...), 28)
			decKey, decIV = short[:16], short[16:28]
			continue
		}
		if decKey == nil {
			continue
		}
		pt, ok := gcmOpen(decKey, ivXor(decIV, i), rec.body, aadOf(i, rec.hdr))
		if ok && rec.typ == 0x17 {
			app = append(app, pt...)
		}
	}
	if len(app) == 0 {
		return nil, false
	}
	return finishResponse(app), true
}

func finishResponse(plain []byte) []byte {
	if i := bytes.Index(plain, []byte("HTTP/")); i >= 0 && i < 32 {
		return inflateHTTP(plain[i:])
	}
	if mostlyText(plain) {
		return plain
	}
	show := plain
	if len(show) > 16 {
		show = show[:16]
	}
	return []byte(fmt.Sprintf("mmtls已解开，响应 %d 字节，没有可读文本。\r\n%s", len(plain), hex.EncodeToString(show)))
}

func mostlyText(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	n := 0
	for _, c := range b {
		if c == '\r' || c == '\n' || c == '\t' || (c >= 32 && c < 127) {
			n++
		}
	}
	return n*100/len(b) >= 85
}

func matchBizText(cgi string) []byte {
	if strings.Contains(cgi, "js-operatewxdata") {
		if text := freshFileText("mmtls_biz_api.bin"); usefulBiz(text) {
			return text
		}
	}
	text := freshBizText()
	if len(text) == 0 || !usefulBiz(text) {
		return nil
	}
	name := cgi
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	if name != "" && bytes.Contains(text, []byte(name)) {
		return text
	}
	if strings.Contains(cgi, "wxausrevent") || strings.Contains(cgi, "js-operatewxdata") || strings.Contains(cgi, "newreport") {
		return text
	}
	return nil
}

func usefulBiz(b []byte) bool {
	s := string(b)
	return strings.Contains(s, "wx") || strings.Contains(s, ".html") || strings.Contains(s, "{") || strings.Contains(s, "/cgi-bin/")
}

func inflateHTTP(plain []byte) []byte {
	sep := bytes.Index(plain, []byte("\r\n\r\n"))
	if sep < 0 {
		return plain
	}
	head := plain[:sep]
	body := plain[sep+4:]
	lower := bytes.ToLower(head)
	var out []byte
	var err error
	switch {
	case bytes.Contains(lower, []byte("content-encoding: deflate")):
		out, err = inflateAny(body)
	case bytes.Contains(lower, []byte("content-encoding: gzip")):
		out, err = inflateGzip(body)
	default:
		return plain
	}
	if err != nil || len(out) == 0 {
		return plain
	}
	return append(append(append([]byte{}, head...), []byte("\r\n\r\n")...), out...)
}

func inflateAny(body []byte) ([]byte, error) {
	if out, err := readAll(zlib.NewReader(bytes.NewReader(body))); err == nil && len(out) > 0 {
		return out, nil
	}
	r := flate.NewReader(bytes.NewReader(body))
	defer r.Close()
	return io.ReadAll(r)
}

func inflateGzip(body []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func readAll(r io.ReadCloser, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

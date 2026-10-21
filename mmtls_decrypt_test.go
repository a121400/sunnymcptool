package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"testing"
)

func u16b(n int) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(n))
	return b
}

func rec(typ byte, payload []byte) []byte {
	h := []byte{typ, 0xF1, 0x04}
	h = append(h, u16b(len(payload))...)
	return append(h, payload...)
}

func TestStringsFromKeepsLongJSON(t *testing.T) {
	js := `{"api_name":"qbase_commapi","data":{"qbase_api_name":"tcbapi_call_gateway","qbase_req":"{\"method\":\"POST\",\"headers\":[{\"k\":\"Content-Type\",\"v\":\"application/json\"}],\"path\":\"/miniapp/v2/draw/participate_instant\",\"data\":\"{\\\"activity_id\\\":57,\\\"id_card\\\":\\\"110101199001011234\\\",\\\"black_box\\\":\\\"abcdefghijklmnopqrstuvwxyz\\\"}\"}}"`
	if len(js) < 200 {
		t.Fatalf("fixture too short: %d", len(js))
	}
	raw := append([]byte{0x00, 0x01}, []byte(js)...)
	got := stringsFrom(raw)
	if !bytes.Contains(got, []byte("participate_instant")) || !bytes.Contains(got, []byte("black_box")) {
		t.Fatalf("json truncated: %s", got)
	}
}

func TestMMTLSRoundTrip(t *testing.T) {
	psk, _ := hex.DecodeString("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	hello := []byte("client-hello-packed-0123456789abcdef")
	sum := sha256.Sum256(hello)
	short := hkdfExpandSHA256(psk, append([]byte("early data key expansion"), sum[:]...), 28)
	key, iv := short[:16], short[16:28]
	app := []byte(`{"activity_id":57,"goods_sku_id":1,"id_card":"123456"}`)
	part2 := []byte{0x00, 0x00, 0x00, 0x10, 0x08}

	seal := func(typ byte, pt []byte, seq int) []byte {
		ctLen := len(pt) + 16
		hdr := []byte{typ, 0xF1, 0x04}
		hdr = append(hdr, u16b(ctLen)...)
		ct := gcmSeal(key, ivXor(iv, seq), pt, aadOf(seq, hdr))
		if ct == nil || len(ct) != ctLen {
			t.Fatalf("seal failed")
		}
		return append(hdr, ct...)
	}
	body := append(rec(0x19, hello), seal(0x19, part2, 1)...)
	body = append(body, seal(0x17, app, 2)...)

	pt, ok := decryptWithKeys(body, mmtlsKeys{psk: psk})
	if !ok || !bytes.Equal(pt, app) {
		t.Fatalf("decrypt request ok=%v pt=%q", ok, pt)
	}
}

// decryptWithKeys 供测试绕过文件。
func decryptWithKeys(body []byte, keys mmtlsKeys) ([]byte, bool) {
	recs := splitMMTLS(body)
	if len(recs) == 0 || len(keys.psk) < 16 {
		return nil, false
	}
	packed := recs[0].body
	sum := sha256.Sum256(packed)
	short := hkdfExpandSHA256(keys.psk, append([]byte("early data key expansion"), sum[:]...), 28)
	key, iv := short[:16], short[16:28]
	var app []byte
	seq := 1
	for _, rec := range recs[1:] {
		pt, ok := gcmOpen(key, ivXor(iv, seq), rec.body, aadOf(seq, rec.hdr))
		seq++
		if ok && rec.typ == 0x17 {
			app = append(app, pt...)
		}
	}
	return app, len(app) > 0
}

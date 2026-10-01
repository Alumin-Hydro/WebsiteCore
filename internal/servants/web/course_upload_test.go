package web

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestSignPostPolicyV4(t *testing.T) {
	// 期望值由 openssl dgst -sha256 -mac HMAC 按官方 HMAC 链逐级独立算出
	want := "4f563994732aeb94c48af6cbee990ebf0081d6f6d7f70dc370d6691c5be68afe"
	if got := signPostPolicyV4("sk", "cn-beijing", "20261001", "eyJ0ZXN0IjoxfQ=="); got != want {
		t.Errorf("签名不符: got %s, want %s", got, want)
	}
}

func TestNewPostPolicyV4UsesUTC(t *testing.T) {
	// 北京时间 10-02 01:00 即 UTC 10-01 17:00, 日期必须取 UTC 当天
	now := time.Date(2026, 10, 2, 1, 0, 0, 0, time.FixedZone("CST", 8*3600))
	resp, err := newPostPolicyV4(now, "ak", "sk", "cn-beijing", "bkt", "attachment/course/202610/a.mp4")
	if err != nil {
		t.Fatalf("newPostPolicyV4 err: %s", err)
	}
	if resp.Credential != "ak/20261001/cn-beijing/oss/aliyun_v4_request" {
		t.Errorf("credential 不符: %s", resp.Credential)
	}
	if resp.Date != "20261001T170000Z" {
		t.Errorf("x-oss-date 不符: %s", resp.Date)
	}
	if want := signPostPolicyV4("sk", "cn-beijing", "20261001", resp.Policy); resp.Signature != want {
		t.Errorf("签名应基于 base64 policy: got %s, want %s", resp.Signature, want)
	}

	raw, err := base64.StdEncoding.DecodeString(resp.Policy)
	if err != nil {
		t.Fatalf("policy 不是合法 base64: %s", err)
	}
	var policy struct {
		Expiration string `json:"expiration"`
		Conditions []any  `json:"conditions"`
	}
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatalf("policy 不是合法 JSON: %s", err)
	}
	if policy.Expiration != "2026-10-01T17:10:00.000Z" {
		t.Errorf("expiration 不符: %s", policy.Expiration)
	}
	// 表单里的 x-oss-* 字段必须与 policy 条件逐字一致, 否则 OSS 拒绝
	fields := map[string]string{}
	for _, c := range policy.Conditions {
		if m, ok := c.(map[string]any); ok {
			for k, v := range m {
				fields[k], _ = v.(string)
			}
		}
	}
	for k, want := range map[string]string{
		"bucket":                  "bkt",
		"x-oss-signature-version": resp.SignatureVersion,
		"x-oss-credential":        resp.Credential,
		"x-oss-date":              resp.Date,
	} {
		if fields[k] != want {
			t.Errorf("policy 条件 %s = %q, want %q", k, fields[k], want)
		}
	}
	if resp.SignatureVersion != "OSS4-HMAC-SHA256" {
		t.Errorf("signature version 不符: %s", resp.SignatureVersion)
	}
}

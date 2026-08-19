package operation_setting

import (
	"encoding/base64"
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var (
	PerPayAddress       = ""
	PerPayClientId      = "default"
	PerPayAPIKey        = ""
	PerPayWebhookSecret = ""
)

var (
	perPayClientIdPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{2,63}$`)
	perPaySecretPattern   = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)
)

func ValidatePerPayAddress(value string) error {
	if value == "" {
		return nil
	}
	if value != strings.TrimSpace(value) {
		return errors.New("PerPay 地址不能包含首尾空格")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return errors.New("PerPay 地址必须是有效的 HTTPS 站点地址")
	}
	if (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("PerPay 地址只能填写站点地址，不能包含路径、查询参数或片段")
	}
	return nil
}

func ValidatePerPayClientId(value string) error {
	if value == "" {
		return nil
	}
	if !perPayClientIdPattern.MatchString(value) {
		return errors.New("PerPay Client ID 必须是 3 到 64 位 URL 安全字符")
	}
	return nil
}

func DecodePerPaySecret(value string) ([]byte, error) {
	if !perPaySecretPattern.MatchString(value) {
		return nil, errors.New("PerPay 密钥必须是 32 字节的 base64url 字符串")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(decoded) != 32 || base64.RawURLEncoding.EncodeToString(decoded) != value {
		return nil, errors.New("PerPay 密钥必须是 32 字节的 base64url 字符串")
	}
	return decoded, nil
}

func ValidatePerPaySecret(value string) error {
	if value == "" {
		return nil
	}
	_, err := DecodePerPaySecret(value)
	return err
}

func IsPerPayConfigured() bool {
	return ValidatePerPayAddress(PerPayAddress) == nil && PerPayAddress != "" &&
		ValidatePerPayClientId(PerPayClientId) == nil && PerPayClientId != "" &&
		ValidatePerPaySecret(PerPayAPIKey) == nil && PerPayAPIKey != "" &&
		ValidatePerPaySecret(PerPayWebhookSecret) == nil && PerPayWebhookSecret != ""
}

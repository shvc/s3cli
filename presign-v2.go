package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"
)

const timeFormatV2 = "2006-01-02T15:04:05"

// Presign2 presing a URL with Signed Signature Version 2.
func Presign2(u *url.URL, method, ak, sk string, exp time.Duration) (string, error) {
	expTime := fmt.Sprintf("%d", time.Now().Add(exp).Unix())
	q := u.Query()
	q.Set("AWSAccessKeyId", ak)
	q.Set("Expires", expTime)
	contentType := ""
	strToSign := fmt.Sprintf("%s\n%s\n%s\n%s\n%s", method, "", contentType, expTime, u.Path)
	signature := signature(strToSign, sk)

	q.Set("Signature", signature)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func signature(strToSign string, sk string) string {
	mac := hmac.New(sha1.New, []byte(sk))
	mac.Write([]byte(strToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

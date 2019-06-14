package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const timeFormatV2 = "2006-01-02T15:04:05"

func augmentRequestQuery(request *http.Request, values url.Values) *http.Request {
	for key, array := range request.URL.Query() {
		for _, value := range array {
			values.Set(key, value)
		}
	}

	request.URL.RawQuery = values.Encode()

	return request
}

func prepareRequestV2(request *http.Request, ak string) *http.Request {
	values := url.Values{}
	values.Set("AWSAccessKeyId", ak)
	values.Set("SignatureVersion", "2")
	values.Set("SignatureMethod", "HmacSHA256")
	values.Set("Timestamp", timestampV2())

	augmentRequestQuery(request, values)

	if request.URL.Path == "" {
		request.URL.Path += "/"
	}

	return request
}

func stringToSignV2(request *http.Request) string {
	str := request.Method + "\n"
	str += strings.ToLower(request.URL.Host) + "\n"
	str += request.URL.Path + "\n"
	str += canonicalQueryStringV2(request)
	return str
}

func hmacSHA256(key []byte, content string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(content))
	return mac.Sum(nil)
}

func signatureV2(strToSign string, sk string) string {
	hashed := hmacSHA256([]byte(sk), strToSign)
	return base64.StdEncoding.EncodeToString(hashed)
}

func canonicalQueryStringV2(request *http.Request) string {
	return request.URL.RawQuery
}

func timestampV2() string {
	return time.Now().UTC().Format(timeFormatV2)
}

package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// AWS Signature Version 2 uses SHA-1 HMAC as specified in the AWS S3 API documentation.
// Note: SHA-1 is used here because it is required by the AWS Signature V2 protocol specification.
// For new applications, prefer AWS Signature Version 4 which uses SHA-256.
// See: https://docs.aws.amazon.com/AmazonS3/latest/userguide/RESTAuthentication.html
// The subresources that must be included when constructing the CanonicalizedResource Element are
// acl, lifecycle, location, logging, notification, partNumber, policy, requestPayment, uploadId,
// uploads, versionId, versioning, versions, and website.

// The delete query string parameter must be included when you create the CanonicalizedResource for a multi-object Delete request.
// URL parameters that need to be added to the signature
var s3ParamsToSign = map[string]struct{}{
	"acl":                          {},
	"delete":                       {},
	"location":                     {},
	"logging":                      {},
	"notification":                 {},
	"partNumber":                   {},
	"policy":                       {},
	"requestPayment":               {},
	"torrent":                      {},
	"uploadId":                     {},
	"uploads":                      {},
	"versionId":                    {},
	"versioning":                   {},
	"versions":                     {},
	"response-content-type":        {},
	"response-content-language":    {},
	"response-expires":             {},
	"response-cache-control":       {},
	"response-content-disposition": {},
	"response-content-encoding":    {},
}

// presignV2 presign URL with escaped key(Object name).
func v2Presign(AccessKey, SecretKey string, expireTime time.Duration, req *http.Request, debug int) {
	exp := strconv.FormatInt(time.Now().Unix()+int64(expireTime.Seconds()), 10)

	q := req.URL.Query()
	q.Set("AWSAccessKeyId", AccessKey)
	q.Set("Expires", exp)
	contentType := req.Header.Get("Content-Type")

	contentMd5 := req.Header.Get("Content-MD5")
	strToSign := fmt.Sprintf("%s\n%s\n%s\n%v\n%s", req.Method, contentMd5, contentType, exp, req.URL.EscapedPath())
	logSigningInfo(strToSign, debug)

	mac := hmac.New(sha1.New, []byte(SecretKey))
	if _, err := mac.Write([]byte(strToSign)); err != nil {
		// HMAC.Write never returns an error, but check for completeness
		return
	}

	q.Set("Signature", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	req.URL.RawQuery = q.Encode()
}

// sign signs requests using v2 auth
//
// Cobbled together from goamz and aws-sdk-go
func sign(AccessKey, SecretKey string, req *http.Request, debug int) {
	// Set date
	date := time.Now().UTC().Format(time.RFC1123)
	req.Header.Set("Date", date)

	// Sort out URI
	uri := req.URL.EscapedPath()
	if uri == "" {
		uri = "/"
	}

	// Look through headers of interest
	var md5 string
	var contentType string
	var headersToSign []string
	tmpHeadersToSign := make(map[string][]string)
	for k, v := range req.Header {
		k = strings.ToLower(k)
		switch k {
		case "content-md5":
			md5 = v[0]
		case "content-type":
			contentType = v[0]
		default:
			if strings.HasPrefix(k, "x-amz-") {
				tmpHeadersToSign[k] = v
			}
		}
	}
	var keys []string
	for k := range tmpHeadersToSign {
		keys = append(keys, k)
	}
	// https://docs.aws.amazon.com/AmazonS3/latest/dev/RESTAuthentication.html
	sort.Strings(keys)

	for _, key := range keys {
		vall := strings.Join(tmpHeadersToSign[key], ",")
		headersToSign = append(headersToSign, key+":"+vall)
	}
	// Make headers of interest into canonical string
	var joinedHeadersToSign string
	if len(headersToSign) > 0 {
		joinedHeadersToSign = strings.Join(headersToSign, "\n") + "\n"
	}

	// Look for query parameters which need to be added to the signature
	params := req.URL.Query()
	var queriesToSign []string
	for k, vs := range params {
		if _, ok := s3ParamsToSign[k]; ok {
			for _, v := range vs {
				if v == "" {
					queriesToSign = append(queriesToSign, k)
				} else {
					queriesToSign = append(queriesToSign, k+"="+v)
				}
			}
		}
	}
	// Add query parameters to URI
	if len(queriesToSign) > 0 {
		sort.StringSlice(queriesToSign).Sort()
		uri += "?" + strings.Join(queriesToSign, "&")
	}

	// Make signature
	payload := req.Method + "\n" + md5 + "\n" + contentType + "\n" + date + "\n" + joinedHeadersToSign + uri
	logSigningInfo(payload, debug)
	hash := hmac.New(sha1.New, []byte(SecretKey))
	if _, err := hash.Write([]byte(payload)); err != nil {
		// HMAC.Write never returns an error, but check for completeness
		return
	}
	signature := make([]byte, base64.StdEncoding.EncodedLen(hash.Size()))
	base64.StdEncoding.Encode(signature, hash.Sum(nil))

	// Set signature in request
	req.Header.Set("Authorization", "AWS "+AccessKey+":"+string(signature))
}

const logSignInfoMsg = `DEBUG: Request Signature:
---[ STRING TO SIGN ]--------------------------------
%s
-----------------------------------------------------
`

func logSigningInfo(stringToSign string, debug int) {
	if debug < 2 {
		return
	}
	fmt.Printf(logSignInfoMsg, stringToSign)
}

/*

Authorization = "AWS" + " " + AWSAccessKeyId + ":" + Signature;

Signature = Base64( HMAC-SHA1( UTF-8-Encoding-Of(YourSecretAccessKey), UTF-8-Encoding-Of( StringToSign ) ) );

StringToSign = HTTP-Verb + "\n" +
	Content-MD5 + "\n" +
	Content-Type + "\n" +
	Date + "\n" +
	CanonicalizedAmzHeaders +
	CanonicalizedResource;

CanonicalizedResource = [ "/" + Bucket ] +
	<HTTP-Request-URI, from the protocol name up to the query string> +
	[ subresource, if present. For example "?acl", "?location", or "?logging"];

CanonicalizedAmzHeaders = <described below>
*/

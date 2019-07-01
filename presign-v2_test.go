package main

import "testing"

func Test_signatrue(t *testing.T) {

	type sigval struct {
		data string
		key  string
		sig  string
	}
	testData := []sigval{
		sigval{"data", "key", "EEFSxb/coHvGM+69RhmfAlXJ9J0="},
		sigval{"data", "key2", "otRrDCWLu9bV7he/9iXeVEqMBkk="},
	}

	for _, v := range testData {
		sig := signature(v.data, v.key)
		if sig != v.sig {
			t.Error(v.data, v.key, sig)
		}
	}
}

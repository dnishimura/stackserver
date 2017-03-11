package main

import (
	"testing"
)

func Assert(t *testing.T, statement bool, errArgs ...interface{}) {
	if statement == false {
		t.Fatal(errArgs...)
	}
}

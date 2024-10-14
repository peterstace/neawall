package main

import "fmt"

type BadStatusError struct {
	body   []byte
	status int
}

func (b BadStatusError) Error() string {
	prefix := string(b.body[:min(len(b.body), 256)])
	return fmt.Sprintf("unexpected status code %d: %s", b.status, prefix)
}

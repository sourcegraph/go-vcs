package vcs

import "io"

type nopCloser struct {
	io.ReadSeeker
}

func (nc nopCloser) Close() error { return nil }

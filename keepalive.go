package main

import "io"

type keepaliveReader struct {
	r     io.Reader
	alive chan struct{}
}

func (r *keepaliveReader) Read(b []byte) (n int, err error) {
	n, err = r.r.Read(b)
	if n > 0 {
		select {
		case r.alive <- struct{}{}:
		default:
		}
	}
	return
}

type keepaliveWriter struct {
	w     io.Writer
	alive chan struct{}
}

func (w *keepaliveWriter) Write(b []byte) (n int, err error) {
	n, err = w.w.Write(b)
	if n > 0 {
		select {
		case w.alive <- struct{}{}:
		default:
		}
	}
	return
}

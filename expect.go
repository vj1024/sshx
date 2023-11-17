package main

import (
	"io"
	"strings"
	"time"
)

var (
	processStartAt       = time.Now()
	autoExpectPasswordIn = 60 * time.Second
)

type expectWriter struct {
	receive io.Writer
	send    io.Writer
	server  *Server
}

func (r *expectWriter) Write(b []byte) (n int, err error) {
	n, err = r.receive.Write(b)

	if n == 0 || n > 1024 {
		return
	}

	if r.server.expectEnd {
		return
	}

	if !r.server.passwordSent && time.Since(processStartAt) > autoExpectPasswordIn {
		r.server.passwordSent = true
	}

	if r.server.Password != "" && !r.server.passwordSent && isPasswordPrompt(b) {
		//log.Printf("expect `password`, send: `%s`", r.server.Password)
		r.server.passwordSent = true
		r.send.Write([]byte(r.server.Password + "\n"))
		return
	}

	if len(r.server.Expect) == 0 {
		return
	}

	for _, v := range r.server.Expect {
		if v.sendTimes < v.SendMaxTimes && v.match != nil && v.match.Match(b) {
			//log.Printf("expect `%s`, send: `%s`, end: %v", v.Match, v.Send, v.End)
			r.send.Write([]byte(v.Send + "\n"))
			v.sendTimes++
			if v.End {
				r.server.expectEnd = true
			}
			return
		}
	}
	return
}

func isPasswordPrompt(b []byte) bool {
	minLen := len("x@x's password:") // end space is optional
	if len(b) < minLen || len(b) > 1024 {
		return false
	}

	s := string(b[len(b)-minLen:])
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	return strings.HasSuffix(s, " password:")
}

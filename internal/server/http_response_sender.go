package server

import "net/http"

type HttpResponseSender struct {
	w http.ResponseWriter
}

func (s *HttpResponseSender) Allow() {
	s.w.WriteHeader(http.StatusNoContent)
}

func (s *HttpResponseSender) Deny() {
	s.w.Header().Add("Location", "/aegis/token")
	s.w.WriteHeader(http.StatusForbidden)
}

func NewHttpResponseSender(w http.ResponseWriter) *HttpResponseSender {
	return &HttpResponseSender{w: w}
}

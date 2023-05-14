package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleQuote(t *testing.T) {
	// 1. HTTP Recorder erstellen
	recorder := httptest.NewRecorder()

	// 2. Request erstellen (mit Body)
	body := `
	{
		"age": 8,
		"breed": "chow"
	 }
	`
	req, _ := http.NewRequest("GET", "/api/quote", strings.NewReader(body))

	// 3. Handler Funktion aufrufen
	HandleQuote(recorder, req)

	// 4. Return Code prüfen
	if recorder.Code != http.StatusOK {
		t.Errorf("Wrong status: got %v expected %v", recorder.Code, http.StatusOK)
	}
}

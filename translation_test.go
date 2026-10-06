package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTranslationProxy(t *testing.T) {
	const body = `{"text":"Hello","source_language":"auto","target_language":"zh-CN"}`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.RequestURI() != "/custom/translate?version=1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.RequestURI())
		}
		payload, _ := io.ReadAll(r.Body)
		if string(payload) != body || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("request body/header changed: %s", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"translated_text":"你好"}`)
	}))
	defer upstream.Close()
	handler, err := newTranslationHandler(upstream.URL + "/custom/translate?version=1")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/translate", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != `{"translated_text":"你好"}` {
		t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
	}
}

func TestTranslationProxyForwardsServiceError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"error":{"message":"服务繁忙"}}`)
	}))
	defer upstream.Close()
	handler, _ := newTranslationHandler(upstream.URL)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/translate", nil))
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "服务繁忙") {
		t.Fatalf("service error was lost: %d %s", response.Code, response.Body.String())
	}
}

func TestTranslationProxyTimeout(t *testing.T) {
	handler, _ := newTranslationHandler("http://127.0.0.1:8000/v1/translate")
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/translate", nil).WithContext(ctx)
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected timeout, got %d %s", response.Code, response.Body.String())
	}
}

func TestTranslationEndpointConfig(t *testing.T) {
	t.Setenv("LYCHEE_TRANSLATE_URL", "https://translate.example.com/custom/translate")
	if configuredTranslationEndpoint() != "https://translate.example.com/custom/translate" {
		t.Fatal("endpoint override was ignored")
	}
	for _, endpoint := range []string{"", "localhost:8000", "file:///tmp/api"} {
		if _, err := newTranslationHandler(endpoint); err == nil {
			t.Errorf("invalid endpoint accepted: %q", endpoint)
		}
	}
}

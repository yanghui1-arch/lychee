package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

// Set with -ldflags "-X main.translationEndpoint=https://example.com/v1/translate".
var translationEndpoint = "http://127.0.0.1:8000/v1/translate"

func configuredTranslationEndpoint() string {
	if endpoint := os.Getenv("LYCHEE_TRANSLATE_URL"); endpoint != "" {
		return endpoint
	}
	return translationEndpoint
}

func newTranslationHandler(endpoint string) (http.Handler, error) {
	target, err := url.Parse(endpoint)
	if err != nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") {
		return nil, errors.New("LYCHEE_TRANSLATE_URL must be a full HTTP(S) URL")
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(target)
			request.Out.URL.Path = target.Path
			request.Out.URL.RawPath = target.RawPath
			request.Out.URL.RawQuery = target.RawQuery
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			status := http.StatusBadGateway
			message := "无法连接翻译服务，请稍后重试。"
			if errors.Is(err, context.DeadlineExceeded) {
				status = http.StatusGatewayTimeout
				message = "翻译请求超时，请重试。"
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"message": message},
			})
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/translate" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		proxy.ServeHTTP(w, r.WithContext(ctx))
	}), nil
}

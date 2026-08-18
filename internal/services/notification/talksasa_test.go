package notification

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTalksasaSenderSendsDocumentedPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/sms/send" || r.Header.Get("Authorization") != "Bearer token" || r.Header.Get("Accept") != "application/json" {
			t.Fatalf("unexpected Talksasa request: %s %s", r.Method, r.URL.String())
		}
		body, _ := io.ReadAll(r.Body)
		for _, value := range []string{"\"recipient\":\"254700000001\"", "\"sender_id\":\"TALKSASA\"", "\"type\":\"plain\"", "\"message\":\"hello\""} {
			if !strings.Contains(string(body), value) {
				t.Fatalf("request missing %s: %s", value, body)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{}}`))
	}))
	defer server.Close()

	sender, err := NewTalksasaSender(server.URL+"/api/v3", "token", "TALKSASA", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if err := sender.Send(context.Background(), Message{To: "254700000001", Body: "hello"}); err != nil {
		t.Fatal(err)
	}
}

func TestTalksasaHashedRejectsInvalidFormat(t *testing.T) {
	sender, err := NewTalksasaSender("http://localhost", "token", "TALKSASA", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := sender.SendHashed(context.Background(), "abc", "invalid", "hello"); err == nil {
		t.Fatal("invalid recipient format was accepted")
	}
}

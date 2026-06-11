package lbrycrd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRawRequestUsesBoundedHTTPTransport(t *testing.T) {
	originalURL := LBRYcrdURL
	originalTimeout := DefaultClientTimeout
	defer func() {
		LBRYcrdURL = originalURL
		DefaultClientTimeout = originalTimeout
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Fatal("expected basic auth")
		}
		if username != "lbry" || password != "secret" {
			t.Fatalf("unexpected credentials %s:%s", username, password)
		}
		var request rpcRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			t.Fatal(err)
		}
		if request.Jsonrpc != "1.0" {
			t.Fatalf("unexpected jsonrpc version %s", request.Jsonrpc)
		}
		if request.Method != "getblockhash" {
			t.Fatalf("unexpected method %s", request.Method)
		}
		if len(request.Params) != 1 {
			t.Fatalf("expected 1 param, got %d", len(request.Params))
		}
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(`{"result":"abc","error":null,"id":1}`))
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	LBRYcrdURL = "rpc://lbry:secret@" + server.Listener.Addr().String()
	DefaultClientTimeout = time.Second

	result, err := callNoDecode("getblockhash", uint64(7))
	if err != nil {
		t.Fatal(err)
	}
	if result != "abc" {
		t.Fatalf("expected abc, got %#v", result)
	}
}

func TestRawRequestTimesOut(t *testing.T) {
	originalURL := LBRYcrdURL
	originalTimeout := DefaultClientTimeout
	defer func() {
		LBRYcrdURL = originalURL
		DefaultClientTimeout = originalTimeout
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	LBRYcrdURL = "rpc://lbry:secret@" + server.Listener.Addr().String()
	DefaultClientTimeout = 20 * time.Millisecond

	start := time.Now()
	_, err := callNoDecode("getblockcount")
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed > 150*time.Millisecond {
		t.Fatalf("expected timeout before handler returned, took %s", elapsed)
	}
}

func TestNewRPCHTTPClientConfiguresTransportTimeouts(t *testing.T) {
	client := newRPCHTTPClient(20 * time.Second)
	if client.Timeout != 20*time.Second {
		t.Fatalf("expected client timeout 20s, got %s", client.Timeout)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected http transport, got %T", client.Transport)
	}
	if transport.TLSHandshakeTimeout != 10*time.Second {
		t.Fatalf("expected TLS handshake timeout 10s, got %s", transport.TLSHandshakeTimeout)
	}
	if transport.ResponseHeaderTimeout != 10*time.Second {
		t.Fatalf("expected response header timeout 10s, got %s", transport.ResponseHeaderTimeout)
	}
	if transport.IdleConnTimeout != 20*time.Second {
		t.Fatalf("expected idle timeout 20s, got %s", transport.IdleConnTimeout)
	}
	if transport.ExpectContinueTimeout != 10*time.Second {
		t.Fatalf("expected expect-continue timeout 10s, got %s", transport.ExpectContinueTimeout)
	}
}

func TestRawRequestRejectsNonSuccessHTTPStatus(t *testing.T) {
	originalURL := LBRYcrdURL
	originalTimeout := DefaultClientTimeout
	defer func() {
		LBRYcrdURL = originalURL
		DefaultClientTimeout = originalTimeout
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, err := w.Write([]byte(`{"result":"abc","error":null,"id":1}`))
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	LBRYcrdURL = "rpc://lbry:secret@" + server.Listener.Addr().String()
	DefaultClientTimeout = time.Second

	_, err := rawRequest("getblockhash", nil)
	if err == nil {
		t.Fatal("expected HTTP status error")
	}
}

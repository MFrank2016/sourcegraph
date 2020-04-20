package bundles

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSendUpload(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("unexpected method. want=%s have=%s", "POST", r.Method)
		}
		if r.URL.Path != "/uploads/42" {
			t.Errorf("unexpected method. want=%s have=%s", "/uploads/42", r.URL.Path)
		}

		if content, err := ioutil.ReadAll(r.Body); err != nil {
			t.Fatalf("unexpected error reading payload: %s", err)
		} else if !reflect.DeepEqual(content, []byte("payload\n")) {
			t.Errorf("unexpected request payload. want=%s have=%s", "payload\n", content)
		}
	}))
	defer ts.Close()

	client := &bundleManagerClientImpl{bundleManagerURL: ts.URL}
	err := client.SendUpload(context.Background(), 42, bytes.NewReader([]byte("payload\n")))
	if err != nil {
		t.Fatalf("unexpected error sending upload: %s", err)
	}
}

func TestSendUploadBadResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := &bundleManagerClientImpl{bundleManagerURL: ts.URL}
	err := client.SendUpload(context.Background(), 42, bytes.NewReader([]byte("payload\n")))
	if err == nil {
		t.Fatalf("unexpected nil error sending upload")
	}
}
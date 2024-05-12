package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	setupHandlers()
	os.Exit(m.Run())
}

func Test_IndexContent(t *testing.T) {
	httpServer, addr := startTestServer(t)

	tests := []struct {
		name    string
		url     string
		content string
		invert  bool
	}{
		{
			name:    "title exists",
			url:     "/",
			content: "<title>CRUDD</title>",
			invert:  false,
		},
		{
			name:    "templates execute",
			url:     "/",
			content: "failed to execute template",
			invert:  true,
		},
		{
			name:    "footer exists",
			url:     "/",
			content: "<div class=\"copyright\">",
			invert:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := queryHttpServer(t, addr+tt.url)

			contains := strings.Contains(response, tt.content)

			if tt.invert {
				if contains {
					t.Fatalf("HTML output was unexpected! Should not contain %q, but did:\n %v", tt.content, response)
				}
			} else {
				if !contains {
					t.Fatalf("HTML output was unexpected! Should contain %q, but did not:\n %v", tt.content, response)
				}
			}
		})
	}

	stopTestServer(t, httpServer)
}

func Test_Command(t *testing.T) {
	httpServer, addr := startTestServer(t)

	tmp, err := os.MkdirTemp(os.TempDir(), t.Name())
	if err != nil {
		t.Fatalf("failed to create tmpdir for test: %v", err)
	}
	defer os.RemoveAll(tmp)

	windows := strings.Contains(tmp, "\\")

	*testFSRoot = tmp

	path := tmp + "/usr/bin"

	var command string
	if windows {
		command = path + "/top.bat"
	} else {
		command = path + "/top.sh"
	}

	if windows {
		path = strings.ReplaceAll(path, "/", "\\")
	}

	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatalf("failed to create command path")
	}

	f, err := os.Create(command)
	if err != nil {
		t.Fatalf("failed to create tmp file for test: %v", err)
	}
	defer os.Remove(f.Name())

	if err := os.Chmod(command, 0700); err != nil {
		t.Fatalf("failed to set executable bit on command")
	}

	var fakeCommandBytes []byte
	if windows {
		fakeCommandBytes = []byte(`@echo off
		echo fake command output`)
	} else {
		fakeCommandBytes = []byte(`echo fake command output`)
	}

	if _, err := f.Write(fakeCommandBytes); err != nil {
		t.Fatalf("failed to write to tmp file for test: %v", err)
	}

	if f.Close(); err != nil {
		t.Fatalf("failed to close tmp file for test: %v", err)
	}

	response := queryHttpServer(t, addr+"/top")

	want := "<pre>fake command output"

	if contains := strings.Contains(response, want); !contains {
		t.Fatalf("HTML output was unexpected! Should contain %q, but did not:\n %v", want, response)
	}

	stopTestServer(t, httpServer)
}

func queryHttpServer(t *testing.T, url string) string {
	t.Helper()

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("http get failed: %v", err)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	return string(bodyBytes)
}

func startTestServer(t *testing.T) (*http.Server, string) {
	t.Helper()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to grab free port: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	log.Printf("found free port: %v", port)

	httpServer := &http.Server{
		Handler: logging()(http.DefaultServeMux),
	}

	go func() {
		log.Println("Starting test CRUDD on port", port)
		if err := httpServer.Serve(listener); err != nil {
			if err != http.ErrServerClosed {
				log.Fatalf("Could not listen on %d: %v\n", port, err)
			}
		}
	}()

	addr := fmt.Sprintf("http://localhost:%d", port)

	return httpServer, addr
}

func stopTestServer(t *testing.T, httpServer *http.Server) {
	t.Helper()

	log.Print("Stopping test CRUDD")
	if err := httpServer.Close(); err != nil {
		t.Fatalf("failed to stop httpServer: %v", err)
	}
}

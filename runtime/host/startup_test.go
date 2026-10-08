package host

import (
	"context"
	"net"
	"net/http"
	"testing"
)

func TestListenerStartupReleasesHTTPWhenMCPBindFails(t *testing.T) {
	available, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	firstAddress := available.Addr().String()
	available.Close()
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	listeners, err := listenHTTPServers([]*http.Server{{Addr: firstAddress}, {Addr: occupied.Addr().String()}})
	if err == nil || len(listeners) != 0 {
		t.Fatalf("partial startup listeners=%v err=%v", listeners, err)
	}
	probe, err := net.Listen("tcp", firstAddress)
	if err != nil {
		t.Fatalf("failed startup leaked first listener: %v", err)
	}
	probe.Close()
}

func TestStartedHostCannotCreateAnotherListenerPair(t *testing.T) {
	testDynamicHost(t, false, false, dynamicHostExtension{verify: func(service *Service, _ string) {
		beforeHTTP, beforeMCP := service.Addresses()
		if err := service.Start(context.Background()); err == nil {
			t.Fatal("started host accepted repeated Start")
		}
		afterHTTP, afterMCP := service.Addresses()
		if beforeHTTP != afterHTTP || beforeMCP != afterMCP {
			t.Fatal("repeated startup changed live listener identities")
		}
	}})
}

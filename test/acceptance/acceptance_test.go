package acceptance_test

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/fixture"
)

const (
	objectStorageMockServer = "127.0.0.1:38001"
)

var (
	pagesBinary = flag.String("gitlab-pages-binary", "../../gitlab-pages", "Path to the gitlab-pages binary")

	httpPort        = "36000"
	httpsPort       = "37000"
	httpProxyPort   = "38000"
	httpProxyV2Port = "39000"

	// TODO: Use TCP port 0 everywhere to avoid conflicts. The binary could output
	// the actual port (and type of listener) for us to read in place of the
	// hardcoded values below.
	listeners = []ListenSpec{
		{"http", "127.0.0.1", httpPort},
		{"http", "::1", httpPort},
		{"https", "127.0.0.1", httpsPort},
		{"https", "::1", httpsPort},
		{"proxy", "127.0.0.1", httpProxyPort},
		{"proxy", "::1", httpProxyPort},
		{"https-proxyv2", "127.0.0.1", httpProxyV2Port},
		{"https-proxyv2", "::1", httpProxyV2Port},
	}

	ipv4Listeners = []ListenSpec{
		listeners[0],
		listeners[2],
		listeners[4],
		listeners[6],
	}

	httpListener         = listeners[0]
	httpsListener        = listeners[2]
	proxyListener        = listeners[4]
	httpsProxyv2Listener = listeners[6]
)

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Short() {
		log.Println("Acceptance tests disabled")
		os.Exit(0)
	}

	if _, err := os.Stat(*pagesBinary); os.IsNotExist(err) {
		log.Fatalf("Couldn't find gitlab-pages binary at %s\n", *pagesBinary)
	}

	if ok := TestCertPool.AppendCertsFromPEM([]byte(fixture.Certificate)); !ok {
		fmt.Println("Failed to load cert!")
	}

	go func() {
		for {
			fmt.Println("checking netstat...")
			cmd := exec.Command("netstat", "-plnut")
			err := cmd.Start()
			if err != nil {
				fmt.Printf("NETSTAT FAILED: %+v\n", err)
			}
			//
			//localhost := "127.0.0.1"
			//_, elements, err := netstat.HasIPConnected(localhost)
			//if err != nil {
			//	log.Printf("netstat.HasIPConnected: %+v\n", err)
			//}
			//
			//for _, e := range elements {
			//	log.Printf("HasIPConnected: %+v", e)
			//}
			//for _, port := range []string{
			//	httpPort,
			//	httpsPort,
			//	httpProxyPort,
			//	httpProxyV2Port,
			//} {
			//
			//	_, elements, err := netstat.HasListeningPort(port)
			//	if err != nil {
			//		log.Printf("netstat.HasListeningPort: %+v\n", err)
			//	}
			//
			//	for _, e := range elements {
			//		log.Printf("HasListeningPort: %+v", e)
			//	}
			//}
			//
			time.Sleep(time.Second)
		}
	}()
	os.Exit(m.Run())
}

func skipUnlessEnabled(t *testing.T, conditions ...string) {
	t.Helper()

	for _, condition := range conditions {
		switch condition {
		case "not-inplace-chroot":
			if os.Getenv("TEST_DAEMONIZE") == "inplace" {
				t.Log("Not supported with -daemon-inplace-chroot")
				t.SkipNow()
			}
		default:
			t.Error("Unknown condition:", condition)
			t.FailNow()
		}
	}
}

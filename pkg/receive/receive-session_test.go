package receive

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/v2/pkg/common/dto"
)

// recordingTransport answers every request with the given body and records
// the URLs it was asked for, so tests can assert how FetchFileWithProgressBar
// builds the download URL without opening a real network listener.
type recordingTransport struct {
	body string
	urls []string
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.urls = append(rt.urls, req.URL.String())
	return &http.Response{
		StatusCode:    http.StatusOK,
		Body:          io.NopCloser(strings.NewReader(rt.body)),
		ContentLength: int64(len(rt.body)),
		Request:       req,
	}, nil
}

func newTestReceiveSession(baseUrl string, transport *recordingTransport) *ReceiveSession {
	return &ReceiveSession{
		baseUrl:    &baseUrl,
		identifier: "synco-test",
		httpClient: &http.Client{Transport: transport},
	}
}

// silentProgressbar returns a progressbar that FetchFileWithProgressBar can
// write to without printing anything: with Total == 0, pterm's Add is a no-op.
func silentProgressbar() *pterm.ProgressbarPrinter {
	return &pterm.ProgressbarPrinter{}
}

// fetchAndAssertURL runs FetchFileWithProgressBar for the given entry and
// asserts that exactly one request was made, to wantUrl.
func fetchAndAssertURL(t *testing.T, baseUrl string, fileName string, entry dto.PublicFilesIndexEntry, wantUrl string) {
	t.Helper()
	transport := &recordingTransport{body: "file content"}
	rs := newTestReceiveSession(baseUrl, transport)

	buf, err := rs.FetchFileWithProgressBar(fileName, entry, silentProgressbar())
	if err != nil {
		t.Fatalf("FetchFileWithProgressBar: %v", err)
	}
	if got := buf.String(); got != "file content" {
		t.Errorf("unexpected content: got %q, want %q", got, "file content")
	}
	if len(transport.urls) != 1 || transport.urls[0] != wantUrl {
		t.Errorf("unexpected requests: got %v, want [%s]", transport.urls, wantUrl)
	}
}

// Regression test: an entry whose PublicUri is already a full URL (e.g. an
// S3/CDN target with an absolute baseUri) must be downloaded from that URL
// directly, without prepending the base URL. The old code checked fileName
// (the local file name / index key) instead of PublicUri for the http(s)
// prefix, so such entries fell into the base-URL branches and produced
// broken URLs.
func TestFetchFileWithProgressBar_AbsolutePublicUri(t *testing.T) {
	fetchAndAssertURL(t,
		"https://origin.example.com/_Resources",
		"persistent/logo.png",
		dto.PublicFilesIndexEntry{
			PublicUri:     "https://cdn.example.com/bucket/persistent/logo.png",
			IsAbsoluteUrl: false,
		},
		"https://cdn.example.com/bucket/persistent/logo.png",
	)
}

// Same regression, for entries flagged IsAbsoluteUrl: a fully-qualified
// PublicUri must win over the IsAbsoluteUrl handling (which would concatenate
// the base URL and the full URL into garbage).
func TestFetchFileWithProgressBar_AbsolutePublicUriWithIsAbsoluteUrlFlag(t *testing.T) {
	fetchAndAssertURL(t,
		"https://origin.example.com/_Resources",
		"persistent/logo.png",
		dto.PublicFilesIndexEntry{
			PublicUri:     "http://cdn.example.com/bucket/persistent/logo.png",
			IsAbsoluteUrl: true,
		},
		"http://cdn.example.com/bucket/persistent/logo.png",
	)
}

// IsAbsoluteUrl entries with a host-relative PublicUri are resolved against
// the base URL with the "/_Resources" suffix stripped.
func TestFetchFileWithProgressBar_IsAbsoluteUrlRelativeUri(t *testing.T) {
	fetchAndAssertURL(t,
		"https://origin.example.com/_Resources",
		"media/site/logo.png",
		dto.PublicFilesIndexEntry{
			PublicUri:     "/media/site/logo.png",
			IsAbsoluteUrl: true,
		},
		"https://origin.example.com/media/site/logo.png",
	)
}

// Relative entries are joined onto the base URL, with the <BASE> placeholder
// removed.
func TestFetchFileWithProgressBar_RelativeUri(t *testing.T) {
	fetchAndAssertURL(t,
		"https://origin.example.com/downloads",
		"css/main.css",
		dto.PublicFilesIndexEntry{
			PublicUri:     "<BASE>/css/main.css",
			IsAbsoluteUrl: false,
		},
		"https://origin.example.com/downloads/css/main.css",
	)
}

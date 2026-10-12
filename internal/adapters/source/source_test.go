package source

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	config "github.com/lnrdll/ymr/internal/domain/config"
)

// recordedRequest is a request as the resolver sent it, before the test
// transport rewrote its host.
type recordedRequest struct {
	URL    string
	Header http.Header
}

// fakeRemote serves files keyed by "host/path" from an httptest server. Its
// transport routes every host, github.com and raw.githubusercontent.com
// included, to that server and records each request.
type fakeRemote struct {
	server *httptest.Server
	files  map[string]string

	mu       sync.Mutex
	requests []recordedRequest
}

func newFakeRemote(t *testing.T, files map[string]string) *fakeRemote {
	t.Helper()

	f := &fakeRemote{files: files}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := f.files[r.Host+r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(f.server.Close)

	return f
}

func (f *fakeRemote) RoundTrip(req *http.Request) (*http.Response, error) {
	f.mu.Lock()
	f.requests = append(f.requests, recordedRequest{URL: req.URL.String(), Header: req.Header.Clone()})
	f.mu.Unlock()

	target, err := url.Parse(f.server.URL)
	if err != nil {
		return nil, err
	}

	out := req.Clone(req.Context())
	out.Host = req.URL.Host
	out.URL.Scheme = target.Scheme
	out.URL.Host = target.Host

	return http.DefaultTransport.RoundTrip(out)
}

func (f *fakeRemote) resolver() resolver {
	return newResolver(&http.Client{Transport: f})
}

func (f *fakeRemote) requestedURLs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	urls := make([]string, 0, len(f.requests))
	for _, r := range f.requests {
		urls = append(urls, r.URL)
	}
	return urls
}

func (f *fakeRemote) onlyRequest(t *testing.T) recordedRequest {
	t.Helper()

	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.requests) != 1 {
		t.Fatalf("expected exactly one request, got %d: %v", len(f.requests), f.requests)
	}
	return f.requests[0]
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertURLs(t *testing.T, got []string, want ...string) {
	t.Helper()

	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("requested URLs:\n got: %v\nwant: %v", got, want)
	}
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got %q", want, err.Error())
	}
}

const specYAML = "templates:\n  - t.yaml\ntargetIds:\n  - dev\n"

func assertSpecLoaded(t *testing.T, cfg *config.SpecConfig, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("LoadSpec err = %v", err)
	}
	if len(cfg.Templates) != 1 || cfg.Templates[0] != "t.yaml" {
		t.Fatalf("Templates = %v, want [t.yaml]", cfg.Templates)
	}
	if len(cfg.TargetIds) != 1 || cfg.TargetIds[0] != "dev" {
		t.Fatalf("TargetIds = %v, want [dev]", cfg.TargetIds)
	}
}

func TestLoadSpec_LocalDirectory_ReadsSpecYAMLInside(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.yaml"), specYAML)

	cfg, err := LoadSpec(dir, "")
	assertSpecLoaded(t, cfg, err)
}

func TestLoadSpec_LocalFile_ReadsThatFile(t *testing.T) {
	t.Parallel()

	specFile := filepath.Join(t.TempDir(), "custom.yaml")
	writeFile(t, specFile, specYAML)

	cfg, err := LoadSpec(specFile, "")
	assertSpecLoaded(t, cfg, err)
}

func TestLoadSpec_LocalDirectoryWithoutSpec_FailsNamingPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	_, err := LoadSpec(dir, "")
	assertErrorContains(t, err, "opening local spec "+filepath.Join(dir, "spec.yaml"))
}

func TestLoadSpec_UnknownLocation_Fails(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "missing")

	_, err := LoadSpec(missing, "")
	assertErrorContains(t, err, "could not determine source type for path: "+missing)
}

func TestLoadSpec_InvalidYAML_Fails(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.yaml"), "templates: [unclosed\n")

	_, err := LoadSpec(dir, "")
	assertErrorContains(t, err, "failed to parse spec YAML")
}

func TestLoadSpec_HTTP_FetchesURL(t *testing.T) {
	t.Parallel()

	remote := newFakeRemote(t, map[string]string{"example.com/specs/spec.yaml": specYAML})

	cfg, err := remote.resolver().LoadSpec("https://example.com/specs/spec.yaml", "")
	assertSpecLoaded(t, cfg, err)
	assertURLs(t, remote.requestedURLs(), "https://example.com/specs/spec.yaml")
}

func TestLoadSpec_HTTPNon200_FailsNamingURL(t *testing.T) {
	t.Parallel()

	remote := newFakeRemote(t, nil)

	_, err := remote.resolver().LoadSpec("https://example.com/spec.yaml", "")
	assertErrorContains(t, err, "fetching spec from https://example.com/spec.yaml: bad response from server for https://example.com/spec.yaml: 404 Not Found")
}

func TestLoadSpec_GitHubForms_FetchRawSpecURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		location string
		wantURL  string
	}{
		{
			name:     "repo root defaults to main",
			location: "https://github.com/u/r",
			wantURL:  "https://raw.githubusercontent.com/u/r/main/spec.yaml",
		},
		{
			name:     "repo root with trailing slash",
			location: "https://github.com/u/r/",
			wantURL:  "https://raw.githubusercontent.com/u/r/main/spec.yaml",
		},
		{
			name:     "tree ref",
			location: "https://github.com/u/r/tree/v1",
			wantURL:  "https://raw.githubusercontent.com/u/r/v1/spec.yaml",
		},
		{
			name:     "https dir at ref",
			location: "https://github.com/u/r/dir@v1",
			wantURL:  "https://raw.githubusercontent.com/u/r/v1/dir/spec.yaml",
		},
		{
			name:     "scheme-less dir at ref",
			location: "github.com/u/r/dir@v1",
			wantURL:  "https://raw.githubusercontent.com/u/r/v1/dir/spec.yaml",
		},
		{
			name:     "scheme-less repo at ref",
			location: "github.com/u/r@v1",
			wantURL:  "https://raw.githubusercontent.com/u/r/v1/spec.yaml",
		},
		{
			// Bug: the blob path is ignored and the repo-root spec.yaml is
			// loaded. Fix task: github-spec-locations.
			name:     "blob loads repo-root spec",
			location: "https://github.com/u/r/blob/v1/dir/custom.yaml",
			wantURL:  "https://raw.githubusercontent.com/u/r/v1/spec.yaml",
		},
		{
			// Bug: a file path in the @ref form is treated as a directory.
			// Fix task: github-spec-locations.
			name:     "file at ref is treated as a directory",
			location: "github.com/u/r/dir/spec.yaml@v1",
			wantURL:  "https://raw.githubusercontent.com/u/r/v1/dir/spec.yaml/spec.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			host, path, _ := strings.Cut(strings.TrimPrefix(tt.wantURL, "https://"), "/")
			remote := newFakeRemote(t, map[string]string{host + "/" + path: specYAML})

			cfg, err := remote.resolver().LoadSpec(tt.location, "")
			assertSpecLoaded(t, cfg, err)
			assertURLs(t, remote.requestedURLs(), tt.wantURL)
		})
	}
}

// Bug: a /tree/<ref>/<dir> URL is not recognised as GitHub and is fetched as
// a plain HTTP page. Fix task: github-spec-locations.
func TestLoadSpec_GitHubTreeDir_FetchedAsPlainHTTP(t *testing.T) {
	t.Parallel()

	remote := newFakeRemote(t, map[string]string{"github.com/u/r/tree/v1/dir": specYAML})

	cfg, err := remote.resolver().LoadSpec("https://github.com/u/r/tree/v1/dir", "")
	assertSpecLoaded(t, cfg, err)
	assertURLs(t, remote.requestedURLs(), "https://github.com/u/r/tree/v1/dir")
}

func TestLoadSpec_GitHubNon200_FailsNamingRawURL(t *testing.T) {
	t.Parallel()

	remote := newFakeRemote(t, nil)

	_, err := remote.resolver().LoadSpec("github.com/u/r@v1", "")
	assertErrorContains(t, err, "fetching spec from GitHub https://raw.githubusercontent.com/u/r/v1/spec.yaml: bad response from server")
}

func assertContent(t *testing.T, got []byte, err error, want string) {
	t.Helper()

	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if string(got) != want {
		t.Fatalf("content = %q, want %q", got, want)
	}
}

func TestLoadSpecTemplate_LocalSpec_ResolvesRelativeToSpec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		location func(dir string) string
	}{
		{name: "spec directory", location: func(dir string) string { return dir }},
		{name: "spec file", location: func(dir string) string { return filepath.Join(dir, "spec.yaml") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "spec.yaml"), specYAML)
			writeFile(t, filepath.Join(dir, "tpl", "t.yaml"), "a: 1\n")

			got, err := LoadSpecTemplate(tt.location(dir), "tpl/t.yaml", "")
			assertContent(t, got, err, "a: 1\n")
		})
	}
}

func TestLoadSpecTemplate_LocalSpec_AbsoluteTemplateReadAsIs(t *testing.T) {
	t.Parallel()

	specDir := t.TempDir()
	writeFile(t, filepath.Join(specDir, "spec.yaml"), specYAML)
	absTemplate := filepath.Join(t.TempDir(), "elsewhere.yaml")
	writeFile(t, absTemplate, "b: 2\n")

	got, err := LoadSpecTemplate(specDir, absTemplate, "")
	assertContent(t, got, err, "b: 2\n")
}

func TestLoadSpecTemplate_LocalSpec_MissingTemplateFailsNamingPath(t *testing.T) {
	t.Parallel()

	specDir := t.TempDir()
	writeFile(t, filepath.Join(specDir, "spec.yaml"), specYAML)

	_, err := LoadSpecTemplate(specDir, "missing.yaml", "")
	assertErrorContains(t, err, filepath.Join(specDir, "missing.yaml"))
}

func TestLoadSpecTemplate_UnknownSpecLocation_Fails(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "missing")

	_, err := LoadSpecTemplate(missing, "t.yaml", "")
	assertErrorContains(t, err, "could not determine source type for path: "+missing)
}

func TestLoadSpecTemplate_RemoteSpecs_ResolveRelativeToSpec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		location string
		wantURL  string
	}{
		{
			name:     "HTTP spec drops file, query and fragment",
			location: "https://example.com/specs/spec.yaml?v=1#top",
			wantURL:  "https://example.com/specs/tpl/t.yaml",
		},
		{
			name:     "GitHub dir at ref",
			location: "github.com/u/r/dir@v1",
			wantURL:  "https://raw.githubusercontent.com/u/r/v1/dir/tpl/t.yaml",
		},
		{
			name:     "GitHub repo root",
			location: "https://github.com/u/r",
			wantURL:  "https://raw.githubusercontent.com/u/r/main/tpl/t.yaml",
		},
		{
			// Bug: templates resolve under the blob file itself. Fix task:
			// github-spec-locations.
			name:     "GitHub blob resolves under the blob file",
			location: "https://github.com/u/r/blob/v1/dir/spec.yaml",
			wantURL:  "https://raw.githubusercontent.com/u/r/v1/dir/spec.yaml/tpl/t.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			host, path, _ := strings.Cut(strings.TrimPrefix(tt.wantURL, "https://"), "/")
			remote := newFakeRemote(t, map[string]string{host + "/" + path: "a: 1\n"})

			got, err := remote.resolver().LoadSpecTemplate(tt.location, "tpl/t.yaml", "")
			assertContent(t, got, err, "a: 1\n")
			assertURLs(t, remote.requestedURLs(), tt.wantURL)
		})
	}
}

func TestLoadSpecTemplate_RemoteTemplateURLInSpec_FetchedDirectly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		template string
		wantURL  string
	}{
		{name: "HTTP URL", template: "https://other.example/t.yaml", wantURL: "https://other.example/t.yaml"},
		{name: "GitHub blob URL", template: "https://github.com/u/r/blob/v1/t.yaml", wantURL: "https://raw.githubusercontent.com/u/r/v1/t.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			specDir := t.TempDir()
			writeFile(t, filepath.Join(specDir, "spec.yaml"), specYAML)
			host, path, _ := strings.Cut(strings.TrimPrefix(tt.wantURL, "https://"), "/")
			remote := newFakeRemote(t, map[string]string{host + "/" + path: "a: 1\n"})

			got, err := remote.resolver().LoadSpecTemplate(specDir, tt.template, "")
			assertContent(t, got, err, "a: 1\n")
			assertURLs(t, remote.requestedURLs(), tt.wantURL)
		})
	}
}

// Bug: a remote spec can make ymr read any absolute path on the local
// filesystem. Fix task: remote-spec-local-template.
func TestLoadSpecTemplate_RemoteSpecAbsoluteTemplate_ReadsLocalFile(t *testing.T) {
	t.Parallel()

	absTemplate := filepath.Join(t.TempDir(), "local.yaml")
	writeFile(t, absTemplate, "secret: 1\n")
	remote := newFakeRemote(t, nil)

	got, err := remote.resolver().LoadSpecTemplate("https://example.com/spec.yaml", absTemplate, "")
	assertContent(t, got, err, "secret: 1\n")
	assertURLs(t, remote.requestedURLs())
}

func TestLoadTemplate_RelativeToWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "tpl", "t.yaml"), "a: 1\n")
	t.Chdir(dir)

	got, err := LoadTemplate("tpl/t.yaml", "")
	assertContent(t, got, err, "a: 1\n")
}

func TestLoadTemplate_MissingFile_FailsNamingPath(t *testing.T) {
	t.Chdir(t.TempDir())

	_, err := LoadTemplate("missing.yaml", "")
	assertErrorContains(t, err, "missing.yaml")
}

// Bug: the scheme-less GitHub form is not recognised for templates and is
// read as a local path. Fix task: github-template-refs.
func TestLoadTemplate_SchemelessGitHubRef_ReadAsLocalPath(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "github.com", "u", "r", "t.yaml@v1"), "local: 1\n")
	t.Chdir(dir)
	remote := newFakeRemote(t, nil)

	got, err := remote.resolver().LoadTemplate("github.com/u/r/t.yaml@v1", "")
	assertContent(t, got, err, "local: 1\n")
	assertURLs(t, remote.requestedURLs())
}

func TestLoadTemplate_RemoteURLs_FetchRawURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		template string
		wantURL  string
	}{
		{name: "HTTP URL", template: "https://example.com/t.yaml", wantURL: "https://example.com/t.yaml"},
		{name: "GitHub blob URL", template: "https://github.com/u/r/blob/v1/dir/t.yaml", wantURL: "https://raw.githubusercontent.com/u/r/v1/dir/t.yaml"},
		{name: "GitHub https file at ref", template: "https://github.com/u/r/dir/t.yaml@v1", wantURL: "https://raw.githubusercontent.com/u/r/v1/dir/t.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			host, path, _ := strings.Cut(strings.TrimPrefix(tt.wantURL, "https://"), "/")
			remote := newFakeRemote(t, map[string]string{host + "/" + path: "a: 1\n"})

			got, err := remote.resolver().LoadTemplate(tt.template, "")
			assertContent(t, got, err, "a: 1\n")
			assertURLs(t, remote.requestedURLs(), tt.wantURL)
		})
	}
}

func TestLoadParams_LocalFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    map[string]any
	}{
		{name: "values", content: "name: web\nreplicas: 2\n", want: map[string]any{"name": "web", "replicas": 2}},
		{name: "empty file gives empty map", content: "", want: map[string]any{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file := filepath.Join(t.TempDir(), "params.yaml")
			writeFile(t, file, tt.content)

			got, err := LoadParams(file, "")
			if err != nil {
				t.Fatalf("LoadParams err = %v", err)
			}
			if got == nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("params = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestLoadParams_Errors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	invalid := filepath.Join(dir, "invalid.yaml")
	writeFile(t, invalid, "name: [unclosed\n")
	missing := filepath.Join(dir, "missing.yaml")

	_, err := LoadParams(invalid, "")
	assertErrorContains(t, err, "unmarshaling params file '"+invalid+"'")

	_, err = LoadParams(missing, "")
	assertErrorContains(t, err, "reading params file '"+missing+"'")
}

func TestLoadValidations_LocalFile(t *testing.T) {
	t.Parallel()

	file := filepath.Join(t.TempDir(), "validations.yaml")
	writeFile(t, file, "- rule: params.replicas > 0\n  message: need replicas\n  targetId: [dev]\n")

	got, err := LoadValidations(file, "")
	if err != nil {
		t.Fatalf("LoadValidations err = %v", err)
	}
	want := []config.Validation{{Rule: "params.replicas > 0", Message: "need replicas", TargetId: []string{"dev"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("validations = %#v, want %#v", got, want)
	}
}

func TestLoadValidations_Errors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	invalid := filepath.Join(dir, "invalid.yaml")
	writeFile(t, invalid, "rule: not-a-list\n")
	missing := filepath.Join(dir, "missing.yaml")

	_, err := LoadValidations(invalid, "")
	assertErrorContains(t, err, "unmarshaling validation file '"+invalid+"'")

	_, err = LoadValidations(missing, "")
	assertErrorContains(t, err, "reading validation file '"+missing+"'")
}

func TestLoadParamsAndValidations_RemoteForms_FetchRawURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		location string
		wantURL  string
	}{
		{name: "HTTP URL", location: "https://example.com/p.yaml", wantURL: "https://example.com/p.yaml"},
		{name: "GitHub blob", location: "https://github.com/u/r/blob/v1/dir/p.yaml", wantURL: "https://raw.githubusercontent.com/u/r/v1/dir/p.yaml"},
		{name: "GitHub repo root", location: "https://github.com/u/r", wantURL: "https://raw.githubusercontent.com/u/r/main/"},
		{name: "GitHub tree ref", location: "https://github.com/u/r/tree/v1", wantURL: "https://raw.githubusercontent.com/u/r/v1/"},
		{name: "GitHub https file at ref", location: "https://github.com/u/r/dir/p.yaml@v1", wantURL: "https://raw.githubusercontent.com/u/r/v1/dir/p.yaml"},
		{name: "GitHub scheme-less file at ref", location: "github.com/u/r/dir/p.yaml@v1", wantURL: "https://raw.githubusercontent.com/u/r/v1/dir/p.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			host, path, _ := strings.Cut(strings.TrimPrefix(tt.wantURL, "https://"), "/")
			remote := newFakeRemote(t, map[string]string{host + "/" + path: "[]\n"})
			r := remote.resolver()

			if _, err := r.LoadParams(tt.location, ""); err == nil {
				t.Fatalf("LoadParams: expected a list to fail decoding into a map")
			}
			if _, err := r.LoadValidations(tt.location, ""); err != nil {
				t.Fatalf("LoadValidations err = %v", err)
			}
			assertURLs(t, remote.requestedURLs(), tt.wantURL, tt.wantURL)
		})
	}
}

func TestFetch_Headers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		location   string
		token      string
		wantAuth   string
		wantAccept string
	}{
		{
			name:       "token sent to raw.githubusercontent.com",
			location:   "github.com/u/r/p.yaml@v1",
			token:      "secret",
			wantAuth:   "Bearer secret",
			wantAccept: "application/vnd.github.v3.raw",
		},
		{
			name:       "token sent to github.com",
			location:   "https://github.com/u/r/tree/v1/p.yaml",
			token:      "secret",
			wantAuth:   "Bearer secret",
			wantAccept: "application/vnd.github.v3.raw",
		},
		{
			name:     "token not sent to other hosts",
			location: "https://example.com/p.yaml",
			token:    "secret",
		},
		{
			name:       "no Authorization with empty token",
			location:   "github.com/u/r/p.yaml@v1",
			wantAccept: "application/vnd.github.v3.raw",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			remote := newFakeRemote(t, nil)

			_, _ = remote.resolver().LoadParams(tt.location, tt.token)
			req := remote.onlyRequest(t)

			if got := req.Header.Values("Authorization"); strings.Join(got, ",") != tt.wantAuth {
				t.Fatalf("Authorization = %q, want %q", got, tt.wantAuth)
			}
			if got := req.Header.Values("Accept"); strings.Join(got, ",") != tt.wantAccept {
				t.Fatalf("Accept = %q, want %q", got, tt.wantAccept)
			}
			if got := req.Header.Get("Cache-Control"); got != "no-cache" {
				t.Fatalf("Cache-Control = %q, want no-cache", got)
			}
		})
	}
}

func TestFetch_Errors(t *testing.T) {
	t.Parallel()

	t.Run("non-200 names URL and status", func(t *testing.T) {
		t.Parallel()

		remote := newFakeRemote(t, nil)

		_, err := remote.resolver().LoadTemplate("https://example.com/t.yaml", "")
		assertErrorContains(t, err, "bad response from server for https://example.com/t.yaml: 404 Not Found")
	})

	t.Run("connection failure names URL", func(t *testing.T) {
		t.Parallel()

		remote := newFakeRemote(t, nil)
		remote.server.Close()

		_, err := remote.resolver().LoadParams("https://example.com/p.yaml", "")
		assertErrorContains(t, err, "reading params file 'https://example.com/p.yaml': failed to fetch remote content from https://example.com/p.yaml")
	})
}

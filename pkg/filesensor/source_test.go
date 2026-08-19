package filesensor_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/filesensor"
	"go.emeland.io/modelsrv/pkg/ingress"
	"go.emeland.io/modelsrv/pkg/model"
)

var _ = Describe("LocalSource", func() {
	It("lists and reads supported files", func() {
		dir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(dir, "a.yaml"), []byte("x"), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "b.json"), []byte("{}"), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "c.txt"), []byte("no"), 0644)).To(Succeed())

		src := filesensor.NewLocalSource(dir)
		metas, err := src.List(context.Background())
		Expect(err).NotTo(HaveOccurred())
		names := make([]string, 0, len(metas))
		for _, m := range metas {
			names = append(names, m.Name)
		}
		Expect(names).To(ConsistOf("a.yaml", "b.json"))

		data, err := src.Read(context.Background(), "a.yaml")
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(Equal("x"))
	})
})

var _ = Describe("HTTPSource", func() {
	It("lists and reads a single URL", func() {
		body := []byte(`{"version":"emeland.io/v1","kind":"Context","spec":{"contextId":"22222222-2222-2222-2222-222222222222","displayName":"HTTP"}}`)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("ETag", `"abc"`)
			w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			_, _ = w.Write(body)
		}))
		defer srv.Close()

		src := filesensor.NewHTTPSource(srv.URL + "/landscape.json")
		metas, err := src.List(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(metas).To(HaveLen(1))
		Expect(metas[0].Name).To(Equal("landscape.json"))
		Expect(metas[0].ETag).To(Equal(`"abc"`))

		data, err := src.Read(context.Background(), "landscape.json")
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(Equal(body))

		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())
		filesensor.ApplySource(context.Background(), src, filesensor.StaticParserConfig{}, m, nil)
		Expect(m.GetContextById(uuid.MustParse("22222222-2222-2222-2222-222222222222"))).NotTo(BeNil())
	})

	It("falls back to Content-Type when the URL has no usable extension", func() {
		body := []byte(`{"version":"emeland.io/v1","kind":"Context","spec":{"contextId":"33333333-3333-3333-3333-333333333333","displayName":"ByContentType"}}`)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_, _ = w.Write(body)
		}))
		defer srv.Close()

		src := filesensor.NewHTTPSource(srv.URL + "/api/landscape")
		metas, err := src.List(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(metas[0].ContentType).To(HavePrefix("application/json"))

		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())
		filesensor.ApplySource(context.Background(), src, filesensor.StaticParserConfig{}, m, nil)
		Expect(m.GetContextById(uuid.MustParse("33333333-3333-3333-3333-333333333333"))).NotTo(BeNil())
	})

	It("uses a client timeout so a hung upstream cannot block forever", func() {
		src := filesensor.NewHTTPSource("https://example.com/landscape.json")
		Expect(src.Client).NotTo(BeNil())
		Expect(src.Client.Timeout).To(Equal(filesensor.DefaultHTTPTimeout))

		hang := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-hang
		}))
		defer func() {
			close(hang)
			srv.Close()
		}()

		src = filesensor.NewHTTPSource(srv.URL + "/landscape.json")
		src.Client = &http.Client{Timeout: 50 * time.Millisecond}
		_, err := src.List(context.Background())
		Expect(err).To(HaveOccurred())
	})

	It("falls back to a ranged GET when HEAD is rejected", func() {
		var gotRange string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			gotRange = r.Header.Get("Range")
			w.Header().Set("ETag", `"abc"`)
			w.Header().Set("Content-Type", "application/json")
			if gotRange == "bytes=0-0" {
				w.Header().Set("Content-Range", "bytes 0-0/1048576")
				w.WriteHeader(http.StatusPartialContent)
				_, _ = w.Write([]byte("{"))
				return
			}
			_, _ = w.Write(bytes.Repeat([]byte("x"), 1<<20))
		}))
		defer srv.Close()

		src := filesensor.NewHTTPSource(srv.URL + "/landscape.json")
		metas, err := src.List(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(gotRange).To(Equal("bytes=0-0"))
		Expect(metas).To(HaveLen(1))
		Expect(metas[0].ETag).To(Equal(`"abc"`))
		Expect(metas[0].ContentType).To(HavePrefix("application/json"))
	})

	It("does not fall back to GET when HEAD returns 404", func() {
		gets := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				gets++
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		src := filesensor.NewHTTPSource(srv.URL + "/missing.json")
		_, err := src.List(context.Background())
		Expect(err).To(MatchError(ContainSubstring("status 404")))
		Expect(gets).To(Equal(0))
	})
})

var _ = Describe("S3Source", func() {
	It("lists and reads flat objects under a prefix", func() {
		client := &filesensor.MemoryS3Client{Objects: map[string]filesensor.MemoryS3Object{
			"landscapes/a.yaml": {
				Data: []byte(`version: emeland.io/v1
kind: Context
spec:
  contextId: "22222222-2222-2222-2222-222222222222"
  displayName: "S3"
`),
				ETag:         "etag-a",
				LastModified: time.Now(),
			},
			"landscapes/nested/b.yaml": {
				Data: []byte("ignored"),
				ETag: "etag-b",
			},
			"landscapes/skip.txt": {Data: []byte("no"), ETag: "e"},
		}}
		src, err := filesensor.NewS3Source("s3://bucket/landscapes/", client)
		Expect(err).NotTo(HaveOccurred())

		metas, err := src.List(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(metas).To(HaveLen(1))
		Expect(metas[0].Name).To(Equal("a.yaml"))

		data, err := src.Read(context.Background(), "a.yaml")
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(ContainSubstring("displayName: \"S3\""))

		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())
		filesensor.ApplySource(context.Background(), src, filesensor.StaticParserConfig{}, m, nil)
		Expect(m.GetContextById(uuid.MustParse("22222222-2222-2222-2222-222222222222"))).NotTo(BeNil())
	})
})

// countingSource is a Source whose contents and change token are controlled by the test.
type countingSource struct {
	mu    sync.Mutex
	etag  string
	data  []byte
	reads int
}

func (s *countingSource) List(ctx context.Context) ([]filesensor.FileMeta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return []filesensor.FileMeta{{Name: "landscape.yaml", ETag: s.etag}}, nil
}

func (s *countingSource) Read(ctx context.Context, name string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reads++
	return append([]byte(nil), s.data...), nil
}

func (s *countingSource) setETag(etag string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.etag = etag
}

func (s *countingSource) readCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reads
}

var _ = Describe("Polling", func() {
	newSource := func() *countingSource {
		return &countingSource{
			etag: "v1",
			data: []byte("version: emeland.io/v1\nkind: Context\nspec:\n  contextId: \"22222222-2222-2222-2222-222222222222\"\n  displayName: Polled\n"),
		}
	}

	It("re-reads only when the change token moves", func() {
		src := newSource()
		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		filesensor.StartSourcePoll(ctx, src, filesensor.StaticParserConfig{}, 20*time.Millisecond, m, nil)

		Eventually(src.readCount).Should(Equal(1))
		Consistently(src.readCount, 200*time.Millisecond).Should(Equal(1))

		src.setETag("v2")
		Eventually(src.readCount).Should(Equal(2))
	})

	It("does not re-apply an already-applied source when StartSources takes over", func() {
		src := newSource()
		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		sources := []filesensor.OpenSource{{
			Config: filesensor.SourceConfig{URI: "s3://bucket/prefix", Poll: 20 * time.Millisecond},
			Source: src,
			Parser: filesensor.StaticParserConfig{},
		}}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		filesensor.ApplySources(ctx, sources, m, nil)
		Expect(src.readCount()).To(Equal(1))
		Expect(m.GetContextById(uuid.MustParse("22222222-2222-2222-2222-222222222222"))).NotTo(BeNil())

		filesensor.StartSources(ctx, sources, m, nil)
		Consistently(src.readCount, 200*time.Millisecond).Should(Equal(1))

		src.setETag("v2")
		Eventually(src.readCount).Should(Equal(2))
	})
})

var _ = Describe("ApplySources", func() {
	openLocal := func(dir string) []filesensor.OpenSource {
		return []filesensor.OpenSource{{
			Source: filesensor.NewLocalSource(dir),
			Parser: filesensor.StaticParserConfig{},
		}}
	}

	It("errors when every document fails to apply", func() {
		dir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(`
version: emeland.io/v1
kind: Context
spec:
  daf: "not-a-context"
`), 0644)).To(Succeed())

		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		summary := filesensor.ApplySources(context.Background(), openLocal(dir), m, nil)
		Expect(summary.Applied).To(Equal(0))
		Expect(summary.Failed).To(BeNumerically(">", 0))
		Expect(summary.Err()).To(MatchError(ContainSubstring("no documents applied")))
	})

	It("errors when every file fails to parse", func() {
		dir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(dir, "broken.yaml"), []byte("{{{not yaml"), 0644)).To(Succeed())

		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		summary := filesensor.ApplySources(context.Background(), openLocal(dir), m, nil)
		Expect(summary.Applied).To(Equal(0))
		Expect(summary.Failed).To(BeNumerically(">", 0))
		Expect(summary.Err()).To(HaveOccurred())
	})

	It("does not error when at least one document applies", func() {
		dir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(dir, "good.yaml"), []byte(`
version: emeland.io/v1
kind: Context
spec:
  contextId: "22222222-2222-2222-2222-222222222222"
  displayName: "Good"
`), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(`
version: emeland.io/v1
kind: Context
spec:
  daf: "nope"
`), 0644)).To(Succeed())

		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		summary := filesensor.ApplySources(context.Background(), openLocal(dir), m, nil)
		Expect(summary.Applied).To(Equal(1))
		Expect(summary.Failed).To(Equal(1))
		Expect(summary.Err()).NotTo(HaveOccurred())
		Expect(m.GetContextById(uuid.MustParse("22222222-2222-2222-2222-222222222222"))).NotTo(BeNil())
	})

	It("does not error when sources contain no files", func() {
		dir := GinkgoT().TempDir()
		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		summary := filesensor.ApplySources(context.Background(), openLocal(dir), m, nil)
		Expect(summary.Applied).To(Equal(0))
		Expect(summary.Failed).To(Equal(0))
		Expect(summary.Err()).NotTo(HaveOccurred())
	})
})

var _ = Describe("Config", func() {
	It("parses multi-source YAML with CSV glob options", func() {
		raw := []byte(`
sources:
  - uri: file:///tmp/data
    watch: true
    files:
      "*.csv":
        format: csv
        delimiter: ","
        columns:
          resourcetype: kind
          uuid: id
          displayname: displayName
`)
		cfg, err := filesensor.ParseConfig(raw)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Sources).To(HaveLen(1))
		Expect(cfg.Sources[0].Watch).To(BeTrue())
		parser := cfg.Sources[0].Files
		Expect(parser["*.csv"].Format).To(Equal("csv"))
		Expect(parser["*.csv"].Columns["uuid"]).To(Equal("id"))
		Expect(parser["*.csv"].Columns["resourcetype"]).To(Equal("kind"))
	})

	It("applies local JSON via ApplyExisting", func() {
		dir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(dir, "c.json"), []byte(`{
  "version": "emeland.io/v1",
  "kind": "Context",
  "spec": {
    "contextId": "22222222-2222-2222-2222-222222222222",
    "displayName": "FromJSON"
  }
}`), 0644)).To(Succeed())

		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())
		filesensor.ApplyExisting(dir, m, nil)
		Expect(m.GetContextById(uuid.MustParse("22222222-2222-2222-2222-222222222222"))).NotTo(BeNil())
	})

	It("applies CSV via GlobParserConfig with per-row resourcetype", func() {
		dir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(dir, "landscape.csv"), []byte(
			"resourcetype,uuid,displayname,description\n"+
				"Context,22222222-2222-2222-2222-222222222222,FromCSV,\n",
		), 0644)).To(Succeed())

		src := filesensor.NewLocalSource(dir)
		cfg := filesensor.GlobParserConfig{Rules: []filesensor.GlobRule{{
			Pattern: "*.csv",
			Opts: ingress.ParseOptions{
				Format: ingress.FormatCSV,
				Columns: map[string]string{
					"resourcetype": "kind",
					"uuid":         "id",
					"displayname":  "displayName",
					"description":  "description",
				},
			},
		}}}
		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())
		filesensor.ApplySource(context.Background(), src, cfg, m, nil)
		Expect(m.GetContextById(uuid.MustParse("22222222-2222-2222-2222-222222222222"))).NotTo(BeNil())
	})

	It("resolves overlapping globs the same way on every open", func() {
		dir := GinkgoT().TempDir()
		cfg, err := filesensor.ParseConfig([]byte(`
sources:
  - uri: file://` + dir + `
    files:
      "*.csv":
        format: csv
        delimiter: ";"
      "landscape.csv":
        format: csv
        delimiter: ","
`))
		Expect(err).NotTo(HaveOccurred())

		// The literal pattern must win over the wildcard, regardless of map order.
		for i := 0; i < 20; i++ {
			_, parser, err := cfg.Sources[0].Open(context.Background())
			Expect(err).NotTo(HaveOccurred())
			opts, err := parser.OptionsFor("landscape.csv")
			Expect(err).NotTo(HaveOccurred())
			Expect(opts.Delimiter).To(Equal(','))

			opts, err = parser.OptionsFor("other.csv")
			Expect(err).NotTo(HaveOccurred())
			Expect(opts.Delimiter).To(Equal(';'))
		}
	})

	It("rejects an unusable parser config instead of silently ignoring it", func() {
		cfg, err := filesensor.ParseConfig([]byte(`
sources:
  - uri: file:///tmp/data
    files:
      "*.csv":
        format: csv
        kind: Contxt
`))
		Expect(err).NotTo(HaveOccurred())
		_, _, err = cfg.Sources[0].Open(context.Background())
		Expect(err).To(MatchError(ContainSubstring("unknown kind")))

		cfg, err = filesensor.ParseConfig([]byte(`
sources:
  - uri: file:///tmp/data
    files:
      "*.xml":
        format: xml
`))
		Expect(err).NotTo(HaveOccurred())
		_, _, err = cfg.Sources[0].Open(context.Background())
		Expect(err).To(MatchError(ContainSubstring("unknown format")))
	})

	It("opens a Source per URI scheme and rejects unknown schemes", func() {
		dir := GinkgoT().TempDir()
		local, _, err := filesensor.SourceConfig{URI: "file://" + dir}.Open(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(local).To(BeAssignableToTypeOf(&filesensor.LocalSource{}))

		bare, _, err := filesensor.SourceConfig{URI: dir}.Open(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(bare).To(BeAssignableToTypeOf(&filesensor.LocalSource{}))

		remote, _, err := filesensor.SourceConfig{URI: "https://example.com/landscape.json"}.Open(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(remote).To(BeAssignableToTypeOf(&filesensor.HTTPSource{}))
		Expect(remote.(*filesensor.HTTPSource).Client.Timeout).To(Equal(filesensor.DefaultHTTPTimeout))

		custom, _, err := filesensor.SourceConfig{URI: "https://example.com/landscape.json", Timeout: 5 * time.Second}.Open(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(custom.(*filesensor.HTTPSource).Client.Timeout).To(Equal(5 * time.Second))

		_, _, err = filesensor.SourceConfig{URI: "ftp://example.com/data"}.Open(context.Background())
		Expect(err).To(MatchError(ContainSubstring("unsupported source scheme")))

		_, _, err = filesensor.SourceConfig{URI: "  "}.Open(context.Background())
		Expect(err).To(MatchError(ContainSubstring("uri is required")))
	})

	It("requires a local source directory to already exist", func() {
		missing := filepath.Join(GinkgoT().TempDir(), "emealnd", "data")
		_, _, err := filesensor.SourceConfig{URI: "file://" + missing}.Open(context.Background())
		Expect(err).To(MatchError(ContainSubstring("does not exist")))
		Expect(err.Error()).To(ContainSubstring(missing))
		_, statErr := os.Stat(missing)
		Expect(os.IsNotExist(statErr)).To(BeTrue())

		_, _, err = filesensor.SourceConfig{URI: missing}.Open(context.Background())
		Expect(err).To(MatchError(ContainSubstring("does not exist")))
		_, statErr = os.Stat(missing)
		Expect(os.IsNotExist(statErr)).To(BeTrue())

		filePath := filepath.Join(GinkgoT().TempDir(), "not-a-dir")
		Expect(os.WriteFile(filePath, []byte("x"), 0644)).To(Succeed())
		_, _, err = filesensor.SourceConfig{URI: "file://" + filePath}.Open(context.Background())
		Expect(err).To(MatchError(ContainSubstring("not a directory")))
	})
})

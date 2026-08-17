package filesensor

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// S3ObjectAPI is the subset of S3 operations used by [S3Source].
// Production code wires AWS SDK clients; tests inject fakes.
type S3ObjectAPI interface {
	ListObjects(ctx context.Context, bucket, prefix string) ([]S3ObjectMeta, error)
	GetObject(ctx context.Context, bucket, key string) ([]byte, error)
}

// S3ObjectMeta is one object returned by ListObjects.
type S3ObjectMeta struct {
	Key          string
	ETag         string
	LastModified time.Time
	Size         int64
}

// S3Source lists and reads objects under an s3://bucket/prefix/ URI.
type S3Source struct {
	Bucket string
	Prefix string
	Client S3ObjectAPI
}

// ParseS3URI parses s3://bucket/prefix into bucket and prefix (prefix may be empty).
func ParseS3URI(raw string) (bucket, prefix string, err error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", err
	}
	if u.Scheme != "s3" {
		return "", "", fmt.Errorf("not an s3 URI: %q", raw)
	}
	bucket = u.Host
	if bucket == "" {
		return "", "", fmt.Errorf("s3 URI missing bucket: %q", raw)
	}
	prefix = strings.TrimPrefix(u.Path, "/")
	return bucket, prefix, nil
}

// NewS3Source returns a Source for s3://bucket/prefix using client.
func NewS3Source(rawURI string, client S3ObjectAPI) (*S3Source, error) {
	bucket, prefix, err := ParseS3URI(rawURI)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("S3 client is required")
	}
	return &S3Source{Bucket: bucket, Prefix: prefix, Client: client}, nil
}

// List implements [Source].
func (s *S3Source) List(ctx context.Context) ([]FileMeta, error) {
	objs, err := s.Client.ListObjects(ctx, s.Bucket, s.Prefix)
	if err != nil {
		return nil, err
	}
	out := make([]FileMeta, 0, len(objs))
	for _, o := range objs {
		name := o.Key
		if s.Prefix != "" {
			name = strings.TrimPrefix(o.Key, s.Prefix)
			name = strings.TrimPrefix(name, "/")
		}
		// v1: flat listing under prefix only (no nested dirs).
		if name == "" || strings.Contains(name, "/") {
			continue
		}
		if !isSupportedFileName(name) {
			continue
		}
		out = append(out, FileMeta{
			Name:         name,
			ETag:         strings.Trim(o.ETag, `"`),
			LastModified: o.LastModified,
		})
	}
	return out, nil
}

// Read implements [Source].
func (s *S3Source) Read(ctx context.Context, name string) ([]byte, error) {
	if filepathBaseInvalid(name) {
		return nil, fmt.Errorf("invalid object name %q", name)
	}
	key := name
	if s.Prefix != "" {
		key = strings.TrimSuffix(s.Prefix, "/") + "/" + name
	}
	return s.Client.GetObject(ctx, s.Bucket, key)
}

func filepathBaseInvalid(name string) bool {
	return name == "" || name == "." || name == ".." || strings.Contains(name, "/") || strings.Contains(name, `\`)
}

// MemoryS3Client is an in-memory [S3ObjectAPI] for tests.
type MemoryS3Client struct {
	Objects map[string]MemoryS3Object // key -> object
}

// MemoryS3Object is one in-memory S3 object.
type MemoryS3Object struct {
	Data         []byte
	ETag         string
	LastModified time.Time
}

// ListObjects implements [S3ObjectAPI].
func (m *MemoryS3Client) ListObjects(ctx context.Context, bucket, prefix string) ([]S3ObjectMeta, error) {
	_ = ctx
	_ = bucket
	var out []S3ObjectMeta
	for k, o := range m.Objects {
		if prefix != "" && !strings.HasPrefix(k, prefix) {
			continue
		}
		out = append(out, S3ObjectMeta{
			Key:          k,
			ETag:         o.ETag,
			LastModified: o.LastModified,
			Size:         int64(len(o.Data)),
		})
	}
	return out, nil
}

// GetObject implements [S3ObjectAPI].
func (m *MemoryS3Client) GetObject(ctx context.Context, bucket, key string) ([]byte, error) {
	_ = ctx
	_ = bucket
	o, ok := m.Objects[key]
	if !ok {
		return nil, fmt.Errorf("s3 object not found: %s", key)
	}
	return append([]byte(nil), o.Data...), nil
}

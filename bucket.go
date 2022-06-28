package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/vanhtuan0409/s3static/assests"
)

var errTryFiles = errors.New("try files error")

type bucket struct {
	name   string
	policy BucketPolicy
	client *minio.Client
}

func NewBucket(client *minio.Client, policy BucketPolicy) *bucket {
	return &bucket{
		name:   policy.Bucket,
		policy: policy,
		client: client,
	}
}

func (b *bucket) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// try exact file path, index file, error file
	isDir := r.URL.Path == "" || strings.HasSuffix(r.URL.Path, "/")
	tryFiles := func() []string {
		if isDir {
			return []string{normalizeS3Path(r.URL.Path, b.policy.IndexDocument)}
		}
		return []string{normalizeS3Path(r.URL.Path)}
	}()
	if b.policy.ErrorDocument != "" {
		tryFiles = append(tryFiles, normalizeS3Path(b.policy.ErrorDocument))
	}

	// perform try files
	err := b.tryFiles(ctx, w, []string{"NoSuchKey"}, tryFiles...)
	if err == nil || err != errTryFiles {
		return
	}

	// render directory listing if allowed as a fallback mechanism
	if isDir && b.policy.AllowListing {
		directoryPath := normalizeS3Path(r.URL.Path)
		err := b.renderDirectory(ctx, w, directoryPath)
		if err != nil {
			responseSimple(w, http.StatusInternalServerError, "internal error")
			log.Printf("render directory error. Bucket: %s, path: %s, err: %+v", b.name, directoryPath, err)
		}
		return
	}

	responseSimple(w, http.StatusNotFound, "not found")
}

func (b *bucket) tryFiles(ctx context.Context, w http.ResponseWriter, passthrough []string, paths ...string) error {
	for _, path := range paths {
		err := b.serveFile(ctx, w, path)
		if err == nil {
			return nil
		}

		s3Err, passed := handleS3Error(w, err, passthrough)
		if passed {
			continue
		}

		log.Printf("upstream err. Bucket: %s, path: %s, err: %+v", b.name, path, s3Err)
		return err
	}
	return errTryFiles
}

func (b *bucket) serveFile(ctx context.Context, w http.ResponseWriter, path string) error {
	obj, err := b.client.GetObject(ctx, b.name, path, minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	defer obj.Close()

	objStat, err := obj.Stat()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", objStat.ContentType)
	w.Header().Set("ETag", objStat.ETag)
	lmt := objStat.LastModified.Truncate(time.Second)
	if !lmt.IsZero() || lmt.Equal(time.Unix(0, 0)) {
		w.Header().Set("Last-Modified", objStat.LastModified.UTC().Format(http.TimeFormat))
	}
	io.Copy(w, obj)
	return nil
}

func (b *bucket) renderDirectory(ctx context.Context, w http.ResponseWriter, path string) error {
	objCh := b.client.ListObjects(ctx, b.name, minio.ListObjectsOptions{
		Prefix: path,
	})

	entries := []*assests.DirectoryEntry{}
	for obj := range objCh {
		if obj.Key == "" {
			continue
		}
		entryName := strings.TrimPrefix(obj.Key, path)
		isDir := strings.HasSuffix(entryName, "/")
		entries = append(entries, &assests.DirectoryEntry{
			Name:  entryName,
			IsDir: isDir,
			Href:  fmt.Sprintf("/%s", obj.Key),
		})
	}

	w.Header().Set("Content-Type", "text/html")
	return assests.RenderDirectory(w, b.name, path, entries)
}

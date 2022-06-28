package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/vanhtuan0409/s3static/assests"
)

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

	// try handle index
	isDir := r.URL.Path == "" || strings.HasSuffix(r.URL.Path, "/")
	tryIndexPath := func() string {
		if isDir {
			return normalizeS3Path(r.URL.Path, b.policy.IndexDocument)
		}
		return normalizeS3Path(r.URL.Path)
	}()

	// allow NoSuchKey to be handle further
	// otherwise stop processing (either success or s3 failure)
	err := b.serveFile(ctx, w, tryIndexPath)
	if s3Err, passed := handleS3Error(w, err, []string{"NoSuchKey"}); !passed {
		if s3Err.Code != "" {
			log.Printf("upstream err. Bucket: %s, path: %s, err: %+v", b.name, tryIndexPath, s3Err)
		}
		return
	}

	// render error document if specified
	if b.policy.ErrorDocument != "" {
		errorPath := normalizeS3Path(b.policy.ErrorDocument)
		err := b.serveFile(ctx, w, errorPath)
		s3Err, _ := handleS3Error(w, err, []string{})
		if s3Err.Code != "" {
			log.Printf("get error document err. Bucket: %s, path: %s, err: %+v", b.name, errorPath, s3Err)
		}
		return
	}

	// render directory listing if allowed
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

package main

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"
)

func normalizeS3Path(paths ...string) string {
	joined := strings.TrimPrefix(path.Join(paths...), "/")
	lpath := len(paths)
	if lpath > 0 {
		lastElem := paths[lpath-1]
		if strings.HasSuffix(lastElem, "/") {
			joined = fmt.Sprintf("%s/", joined)
		}
	}
	return joined
}

func responseSimple(w http.ResponseWriter, status int, format string, v ...any) {
	w.WriteHeader(status)
	fmt.Fprintf(w, format, v...)
}

func handleS3Error(w http.ResponseWriter, err error, passthroughs []string) (s3Err minio.ErrorResponse, passed bool) {
	s3Err = minio.ToErrorResponse(err)
	if s3Err.Message == "" {
		return s3Err, false
	}

	// allow pass through error
	for _, code := range passthroughs {
		if s3Err.Code == code {
			return s3Err, true
		}
	}

	w.WriteHeader(s3Err.StatusCode)
	fmt.Fprintf(w, s3Err.Message)
	return s3Err, false
}

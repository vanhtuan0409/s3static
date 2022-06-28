package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	configFile string
	debug      bool
	s3debug    bool
	dumpAlias  bool
)

func main() {
	flag.StringVar(&configFile, "config", "config.yaml", "Path to config file")
	flag.BoolVar(&debug, "debug", false, "Debug mode")
	flag.BoolVar(&s3debug, "s3debug", false, "S3 Debug mode")
	flag.BoolVar(&dumpAlias, "dump-alias", false, "Dump alias mapping")
	flag.Parse()

	ctx := context.Background()
	conf, err := ParseConfig(ctx, configFile)
	if err != nil {
		log.Fatalf("Cannot read config file or invalid fields. ERR: %+v", err)
	}

	matcher := NewDomainMatcher(conf)
	if dumpAlias {
		matcher.DumpAlias(os.Stdout)
		return
	}

	client, err := minio.New(conf.Endpoint, &minio.Options{
		Creds: credentials.NewChainCredentials([]credentials.Provider{
			&credentials.EnvAWS{},
			&credentials.Static{
				Value: credentials.Value{
					AccessKeyID:     conf.AccessKey,
					SecretAccessKey: conf.SecretKey,
					SignerType:      credentials.SignatureV4,
				},
			},
		}),
		Secure: conf.Secure,
	})
	if err != nil {
		log.Fatalf("Cannot create minio client. ERR: %+v", err)
	}
	if s3debug {
		client.TraceOn(os.Stdout)
	}

	addr := fmt.Sprintf(":%d", conf.HttpPort)
	log.Printf("HTTP server running at %s", addr)
	http.ListenAndServe(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		policy, matched := matcher.Match(r)
		if !matched {
			responseSimple(w, http.StatusNotFound, "not found")
			return
		}

		bucket := NewBucket(client, policy)
		bucket.ServeHTTP(w, r)
	}))
}

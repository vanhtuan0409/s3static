package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type domainmatcher struct {
	conf    *Config
	buckets map[string]BucketPolicy
	aliases map[string]string
}

func NewDomainMatcher(conf *Config) *domainmatcher {
	m := &domainmatcher{
		conf:    conf,
		buckets: make(map[string]BucketPolicy),
		aliases: make(map[string]string),
	}
	m.buildIndex()
	return m
}

func (m *domainmatcher) Match(r *http.Request) (policy BucketPolicy, found bool) {
	requested := m.extractHost(r)
	policy, found = m.lookupAlias(requested)
	if found {
		return
	}

	for _, domain := range m.conf.Domains {
		rootDomainFQDN := fmt.Sprintf(".%s", domain)
		bucket := strings.TrimSuffix(requested, rootDomainFQDN)
		if bucket == requested {
			continue
		}
		return m.lookupBucket(bucket)
	}

	return BucketPolicy{}, false
}

func (m *domainmatcher) DumpAlias(out io.Writer) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(m.aliases)
}

func (m *domainmatcher) buildIndex() {
	for _, p := range m.conf.Policies {
		// index buckets
		m.buckets[p.Bucket] = p

		// index simple domain alias
		for _, alias := range p.DomainAlias {
			m.aliases[alias] = p.Bucket
		}

		// index bucket alias
		for _, bucketAlias := range p.BucketAlias {
			for _, rootDomain := range m.conf.Domains {
				domainAlias := fmt.Sprintf("%s.%s", bucketAlias, rootDomain)
				m.aliases[domainAlias] = p.Bucket
			}
		}
	}
}

func (m *domainmatcher) lookupAlias(domain string) (BucketPolicy, bool) {
	bucket, ok := m.aliases[domain]
	if !ok {
		return BucketPolicy{}, false
	}
	return m.lookupBucket(bucket)
}

func (m *domainmatcher) extractHost(r *http.Request) string {
	xfh := r.Header.Get("X-Forwarded-Host")
	if xfh != "" {
		return xfh
	}
	return r.Host
}

func (m *domainmatcher) lookupBucket(bucket string) (BucketPolicy, bool) {
	policy, found := m.buckets[bucket]
	if !found {
		policy = BucketPolicy{Bucket: bucket}
	}
	m.patchDefault((&policy))
	return policy, true
}

func (m *domainmatcher) patchDefault(policy *BucketPolicy) {
	if policy.IndexDocument == "" {
		policy.IndexDocument = m.conf.DefaultIndexDocument
	}
	if policy.ErrorDocument == "" {
		policy.ErrorDocument = m.conf.DefaultErrorDocument
	}
}

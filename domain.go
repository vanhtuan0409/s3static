package main

import (
	"fmt"
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
	policy, found = m.lookupAliasDomain(requested)
	if found {
		return
	}

	for _, domain := range m.conf.Domains {
		rootDomainFQDN := fmt.Sprintf(".%s", domain)
		bucket := strings.TrimSuffix(requested, rootDomainFQDN)
		if bucket == requested {
			continue
		}
		return m.lookupAliasBucket(bucket)
	}

	return BucketPolicy{}, false
}

func (m *domainmatcher) buildIndex() {
	for _, p := range m.conf.Policies {
		m.buckets[p.Bucket] = p
		for _, alias := range p.DomainAlias {
			m.aliases[alias] = p.Bucket
		}
		for _, bucketAlias := range p.BucketAlias {
			m.aliases[bucketAlias] = p.Bucket
		}
	}
}

func (m *domainmatcher) lookupAliasDomain(domain string) (BucketPolicy, bool) {
	bucket, ok := m.aliases[domain]
	if !ok {
		return BucketPolicy{}, false
	}
	return m.lookupBucket(bucket)
}

func (m *domainmatcher) lookupAliasBucket(bucketAlias string) (BucketPolicy, bool) {
	bucket, ok := m.aliases[bucketAlias]
	if !ok {
		return m.lookupBucket(bucketAlias)
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

// Copyright 2019 tree xie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// HTTP缓存模块，返回当前对应的缓存状态，是获取中、hit for pass等等。
// 以及对缓存数据压缩、智能匹配返回格式等处理。

package cache

import (
	"net/http"
	"sync"
	"time"
)

const (
	// StatusUnknown unknown status
	StatusUnknown = iota
	// StatusFetching fetching status
	StatusFetching
	// StatusHitForPass hit-for-pass status
	StatusHitForPass
	// StatusCachable cachable status
	StatusCachable
	// StatusPassed pass status
	StatusPassed
)

type (
	// HTTPHeader http header
	HTTPHeader [][]byte
	// HTTPHeaders http headers
	HTTPHeaders []HTTPHeader
	// HTTPData http data
	HTTPData struct {
		// Header
		Headers    HTTPHeaders
		StatusCode int
		GzipBody   []byte
		BrBody     []byte
		RawBody    []byte
	}
	// HTTPCache cache status
	HTTPCache struct {
		mu        sync.Mutex
		status    int
		chans     []chan bool
		data      *HTTPData
		createdAt int
		expiredAt int
	}
)

// NewHTTPHeader new a http header
func NewHTTPHeader(key, value []byte) HTTPHeader {
	header := make([][]byte, 2)
	header[0] = key
	header[1] = value
	return header
}

// NewHTTPHeaders new a http headers
func NewHTTPHeaders(header http.Header, ignoreHeaders ...string) (headers HTTPHeaders) {
	headers = make(HTTPHeaders, 0, 10)
	for key, values := range header {
		ignore := false
		for _, ignoreHeader := range ignoreHeaders {
			if ignoreHeader == key {
				ignore = true
			}
		}
		if ignore {
			continue
		}
		k := []byte(key)
		for _, value := range values {
			v := []byte(value)
			h := NewHTTPHeader(k, v)
			headers = append(headers, h)
		}
	}
	return
}

// NewHTTPCache new a http cache
func NewHTTPCache() *HTTPCache {
	return &HTTPCache{}
}

// Get get http cache
func (hc *HTTPCache) Get() (status int, data *HTTPData) {
	status, done, data := hc.get()
	if done != nil {
		<-done
		status = hc.status
		data = hc.data
	}
	return
}

// get get http cache
func (hc *HTTPCache) get() (status int, done chan bool, data *HTTPData) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	now := int(time.Now().Unix())
	// 如果缓存已过期，设置为StatusUnknown
	if hc.expiredAt != 0 && hc.expiredAt < now {
		hc.status = StatusUnknown
	}
	// 如果是fetching，则相同的请求需要等待完成
	// 通过chan bool返回完成
	if hc.status == StatusFetching {
		done = make(chan bool)
		hc.chans = append(hc.chans, done)
	}

	if hc.status == StatusUnknown {
		hc.status = StatusFetching
		hc.chans = make([]chan bool, 0, 5)
	}

	status = hc.status
	// 为什么需要返回status与data
	// 因为有可能在函数调用完成后，刚好缓存过期了，如果此时不返回status与data
	// 当其它goroutin获取锁之后，有可能刚好重置数据
	if status == StatusCachable {
		data = hc.data
	}
	return
}

// HitForPass set the http cache hit for pass
func (hc *HTTPCache) HitForPass(ttl int) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.expiredAt = int(time.Now().Unix()) + ttl
	hc.status = StatusHitForPass
	for _, ch := range hc.chans {
		ch <- true
	}
}

// Cachable set the http cache cachable
func (hc *HTTPCache) Cachable(ttl, statusCode int, rawBody, gzipBody, brBody []byte) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.createdAt = int(time.Now().Unix())
	hc.expiredAt = hc.createdAt + ttl
	hc.status = StatusCachable
	hc.data = &HTTPData{
		StatusCode: statusCode,
		GzipBody:   gzipBody,
		BrBody:     brBody,
		RawBody:    rawBody,
	}
	for _, ch := range hc.chans {
		ch <- true
	}
}
package cache

import (
	"bytes"
	"sync"

	"../util"
	"../vars"
	"github.com/boltdb/bolt"
)

var rsMap = make(map[string]*RequestStatus)
var rsMutex = sync.Mutex{}

var client *bolt.DB

// ResponseData 记录响应数据
type ResponseData struct {
	CreatedAt  uint32
	StatusCode uint16
	Compress   uint16
	TTL        uint32
	Header     []byte
	Body       []byte
}

const (
	createIndex       = 0
	statusCodeIndex   = 4
	compressIndex     = 6
	ttlIndex          = 8
	headerLengthIndex = 12
	headerIndex       = 14
)

// RequestStatus 请求状态
type RequestStatus struct {
	createdAt    uint32
	ttl          uint32
	status       int
	waitingChans []chan int
}

func initRequestStatus(key string, ttl uint32) *RequestStatus {
	rs := &RequestStatus{
		createdAt: util.GetSeconds(),
		ttl:       ttl,
	}
	rsMap[key] = rs
	return rs
}

func isExpired(rs *RequestStatus) bool {
	if rs.ttl != 0 && util.GetSeconds()-rs.createdAt > uint32(rs.ttl) {
		return true
	}
	return false
}

// GetRequestStatus 获取请求的状态
func GetRequestStatus(key []byte) (int, chan int) {
	rsMutex.Lock()
	defer rsMutex.Unlock()
	var c chan int
	k := string(key)
	rs := rsMap[k]
	status := vars.Fetching
	if rs == nil || isExpired(rs) {
		status = vars.Fetching
		rs = initRequestStatus(k, 0)
		rs.status = status
	} else if rs.status == vars.Fetching {
		status = vars.Waiting
		c = make(chan int, 1)
		rs.waitingChans = append(rs.waitingChans, c)
	} else {
		status = rs.status
	}
	return status, c
}

// triggerWatingRequstAndSetStatus 获取等待中的请求，并设置状态和有效期
func triggerWatingRequstAndSetStatus(key []byte, status int, ttl uint32) {
	rsMutex.Lock()
	defer rsMutex.Unlock()
	k := string(key)
	rs := rsMap[k]
	if rs == nil {
		return
	}
	rs.status = status
	rs.ttl = ttl
	waitingChans := rs.waitingChans
	for _, c := range waitingChans {
		c <- status
		close(c)
	}
	rs.waitingChans = nil
}

// HitForPass 触发等待中的请求，并设置状态为hit for pass
func HitForPass(key []byte, ttl uint32) {
	triggerWatingRequstAndSetStatus(key, vars.HitForPass, ttl)
}

// Cacheable 触发等待中的请求，并设置状态为 cacheable
func Cacheable(key []byte, ttl uint32) {
	triggerWatingRequstAndSetStatus(key, vars.Cacheable, ttl)
}

// InitDB 初始化db
func InitDB(file string) (*bolt.DB, error) {
	if client != nil {
		return client, nil
	}
	db, err := bolt.Open(file, 0600, nil)
	if err != nil {
		return nil, err
	}
	client = db
	return db, nil
}

// InitBucket 初始化bucket
func InitBucket(bucket []byte) error {
	if client == nil {
		return vars.ErrDbNotInit
	}
	return client.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	})
}

// SaveResponseData 保存Response
func SaveResponseData(bucket, key []byte, respData *ResponseData) error {
	// 前四个字节保存创建时间
	// 接着后面两个字节保存ttl
	// 接着后面两个字节保存header的长度
	// 接着是header
	// 最后才是body
	createdAt := respData.CreatedAt
	if createdAt == 0 {
		createdAt = util.GetSeconds()
	}
	uint322b := util.ConvertUint32ToBytes
	uint162b := util.ConvertUint16ToBytes
	header := respData.Header

	s := [][]byte{
		uint322b(createdAt),
		uint162b(respData.StatusCode),
		uint162b(respData.Compress),
		uint322b(respData.TTL),
		uint162b(uint16(len(header))),
		header,
		respData.Body,
	}
	data := bytes.Join(s, []byte(""))
	return Save(bucket, key, data)
}

// GetResponse 获取response
func GetResponse(bucket, key []byte) (*ResponseData, error) {
	data, err := Get(bucket, key)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	b2uint16 := util.ConvertBytesToUint16
	b2uint32 := util.ConvertBytesToUint32
	headerLength := b2uint16(data[headerLengthIndex:headerIndex])
	return &ResponseData{
		CreatedAt:  b2uint32(data[createIndex:statusCodeIndex]),
		StatusCode: b2uint16(data[statusCodeIndex:compressIndex]),
		Compress:   b2uint16(data[compressIndex:ttlIndex]),
		TTL:        b2uint32(data[ttlIndex:headerLengthIndex]),
		Header:     data[headerIndex : headerIndex+headerLength],
		Body:       data[headerIndex+headerLength:],
	}, nil
}

// ClearExpiredResponseData 清除已过期缓存
func ClearExpiredResponseData(bucket []byte) error {
	if client == nil {
		return vars.ErrDbNotInit
	}
	expiredTime := util.GetSeconds()
	return client.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		c := b.Cursor()
		b2uint32 := util.ConvertBytesToUint32
		for k, v := c.First(); k != nil; k, v = c.Next() {
			createdAt := b2uint32(v[createIndex:statusCodeIndex])
			ttl := b2uint32(v[ttlIndex:headerLengthIndex])
			if expiredTime > createdAt+ttl {
				b.Delete(k)
			}
		}
		return nil
	})
}

// Save 保存数据
func Save(bucket, key, buf []byte) error {
	if client == nil {
		return vars.ErrDbNotInit
	}
	return client.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		return b.Put(key, buf)
	})
}

// Get 获取数据
func Get(bucket, key []byte) ([]byte, error) {
	if client == nil {
		return nil, vars.ErrDbNotInit
	}
	var buf []byte
	client.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		buf = b.Get(key)
		return nil
	})
	return buf, nil
}

// GetClient 获取 client
func GetClient() *bolt.DB {
	return client
}

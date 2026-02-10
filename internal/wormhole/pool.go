package wormhole

import "sync"

const bufPoolSize = 32 * 1024 // 32KB

var bufPool = sync.Pool{
	New: func() any {
		return make([]byte, bufPoolSize)
	},
}

// GetBuffer returns a buffer from the pool. Call PutBuffer when done.
func GetBuffer() []byte {
	return bufPool.Get().([]byte)
}

// PutBuffer returns a buffer to the pool.
func PutBuffer(b []byte) {
	if cap(b) >= bufPoolSize {
		bufPool.Put(b[:bufPoolSize])
	}
}

package aria2

import (
	"bytes"
	"sync"
)

func addOptionsAndPosition(params []interface{}, options *map[string]interface{}, position *int) []interface{} {
	if options != nil {
		params = append(params, *options)
	}
	if position != nil {
		params = append(params, *position)
	}
	return params
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func NewBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

func PutBuffer(buf *bytes.Buffer) {
	// See https://golang.org/issue/23199
	const maxSize = 1 << 16
	if buf.Cap() < maxSize { // 对于大Buffer直接丢弃
		buf.Reset()
		bufferPool.Put(buf)
	}
}


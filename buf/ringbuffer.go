package buf

import (
	"container/ring"
	"sync"
)

// RingBuffer represents synchronized ring implementation
type RingBuffer struct {
	r *ring.Ring
	m *sync.RWMutex
}

// New creates new RingBuffer
func New(size int) *RingBuffer {
	return &RingBuffer{
		ring.New(size),
		&sync.RWMutex{},
	}
}

// Add adds item to ring
func (buf *RingBuffer) Add(v interface{}) {
	buf.m.Lock()
	defer buf.m.Unlock()
	buf.r.Value = v
	buf.r = buf.r.Next()
}

// Last returned last item added to the ring
func (buf *RingBuffer) Last() interface{} {
	buf.m.Lock()
	defer buf.m.Unlock()
	return buf.r.Prev().Value
}

// Do executes provided callback on all non-nil items of a ring
func (buf *RingBuffer) Do(f func(interface{})) {
	buf.m.RLock()

	defer buf.m.RUnlock()
	buf.r.Do(func(v interface{}) {
		if nil != v {
			f(v)
		}
	})
}

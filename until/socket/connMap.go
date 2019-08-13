package socket

import (
    "sync"
    "fmt"
    "github.com/orcaman/concurrent-map"
)

// ConnMap is a safe map for server connection management.
type ConnMap struct {
    sync.RWMutex
    m cmap.ConcurrentMap
}

// NewConnMap returns a new ConnMap.
func NewConnMap() *ConnMap {
    return &ConnMap {
        m: cmap.New(),
    }
}

// Get gets a server connection with specified net ID.
func (cm *ConnMap) Get(id int) (*TCPConn, bool) {
    key :=  fmt.Sprint(id)
    tmp, ok := cm.m.Get(key)
    if (ok) {
        sc := tmp.(*TCPConn)
        return sc, ok
    }
    return nil, ok 
}

// Put puts a server connection with specified net ID in map.
func (cm *ConnMap) Put(id int, sc *TCPConn) {
    key :=  fmt.Sprint(id)
    cm.m.Set(key, sc)
 
}

// Remove removes a server connection with specified net ID.
func (cm *ConnMap) Remove(id int) {
    key :=  fmt.Sprint(id)
    cm.m.Remove(key)
    
}

// Size returns map size.
func (cm *ConnMap) Size() int {
    size := cm.m.Count()
    return size
}

// Clear clears all elements in map.
func (cm *ConnMap) Clear() {
    cm.Lock()
    cm.m = cmap.New()
    cm.Unlock()
}

//has elements in map.
func (cm *ConnMap) Has(id int) bool {
    key :=  fmt.Sprint(id)
    return cm.m.Has(key)
}

// IsEmpty tells whether ConnMap is empty.
func (cm *ConnMap) IsEmpty() bool {
    return cm.Size() <= 0
}

//GetAll get all the ConnWrapper
func (cm *ConnMap) GetAll() []*TCPConn {
    cm.RLock()
    defer cm.RUnlock()

    var conns []*TCPConn
    for item := range cm.m.IterBuffered() {
        val := item.Val
        if val == nil {
            continue;
        }
        tmp := val.(*TCPConn)
        conns = append(conns, tmp)    
    }
    return conns
}

package common

import (
    "fmt"
    "sync/atomic"
)


// AtomicUint64 provides atomic uint64 type.
type AtomicUint64 uint64

// NewAtomicUint64 returns an atomic uint64 type.
func NewAtomicUint64(initialValue uint64) *AtomicUint64 {
    a := AtomicUint64(initialValue)
    return &a
}

// Get returns the value of uint64 atomically.
func (a *AtomicUint64) Get() uint64 {
    return uint64(*a)
}

// Set sets the value of uint64 atomically.
func (a *AtomicUint64) Set(newValue uint64) {
    atomic.StoreUint64((*uint64)(a), newValue)
}

// GetAndSet sets new value and returns the old atomically.
func (a *AtomicUint64) GetAndSet(newValue uint64) uint64 {
    for {
        current := a.Get()
        if a.CompareAndSet(current, newValue) {
            return current
        }
    }
}

// CompareAndSet compares uint64 with expected value, if equals as expected
// then sets the updated value, this operation performs atomically.
func (a *AtomicUint64) CompareAndSet(expect, update uint64) bool {
    return atomic.CompareAndSwapUint64((*uint64)(a), expect, update)
}

// GetAndIncrement gets the old value and then increment by 1, this operation
// performs atomically.
func (a *AtomicUint64) GetAndIncrement() uint64 {
    for {
        current := a.Get()
        next := current + 1
        if a.CompareAndSet(current, next) {
            return current
        }
    }

}

// GetAndDecrement gets the old value and then decrement by 1, this operation
// performs atomically.
func (a *AtomicUint64) GetAndDecrement() uint64 {
    for {
        current := a.Get()
        next := current - 1
        if a.CompareAndSet(current, next) {
            return current
        }
    }
}

// GetAndAdd gets the old value and then add by delta, this operation
// performs atomically.
func (a *AtomicUint64) GetAndAdd(delta uint64) uint64 {
    for {
        current := a.Get()
        next := current + delta
        if a.CompareAndSet(current, next) {
            return current
        }
    }
}

// IncrementAndGet increments the value by 1 and then gets the value, this
// operation performs atomically.
func (a *AtomicUint64) IncrementAndGet() uint64 {
    for {
        current := a.Get()
        next := current + 1
        if a.CompareAndSet(current, next) {
            return next
        }
    }
}

// DecrementAndGet decrements the value by 1 and then gets the value, this
// operation performs atomically.
func (a *AtomicUint64) DecrementAndGet() uint64 {
    for {
        current := a.Get()
        next := current - 1
        if a.CompareAndSet(current, next) {
            return next
        }
    }
}

// AddAndGet adds the value by delta and then gets the value, this operation
// performs atomically.
func (a *AtomicUint64) AddAndGet(delta uint64) uint64 {
    for {
        current := a.Get()
        next := current + delta
        if a.CompareAndSet(current, next) {
            return next
        }
    }
}

func (a *AtomicUint64) String() string {
    return fmt.Sprintf("%d", a.Get())
}

// AtomicUint32 provides atomic uint32 type.
type AtomicUint32 uint32

// NewAtomicUint32 returns an atomoic uint32 type.
func NewAtomicUint32(initialValue uint32) *AtomicUint32 {
    a := AtomicUint32(initialValue)
    return &a
}

// Get returns the value of uint32 atomically.
func (a *AtomicUint32) Get() uint32 {
    return uint32(*a)
}

// Set sets the value of uint32 atomically.
func (a *AtomicUint32) Set(newValue uint32) {
    atomic.StoreUint32((*uint32)(a), newValue)
}

// GetAndSet sets new value and returns the old atomically.
func (a *AtomicUint32) GetAndSet(newValue uint32) (oldValue uint32) {
    for {
        oldValue = a.Get()
        if a.CompareAndSet(oldValue, newValue) {
            return
        }
    }
}

// CompareAndSet compares uint32 with expected value, if equals as expected
// then sets the updated value, this operation performs atomically.
func (a *AtomicUint32) CompareAndSet(expect, update uint32) bool {
    return atomic.CompareAndSwapUint32((*uint32)(a), expect, update)
}

// GetAndIncrement gets the old value and then increment by 1, this operation
// performs atomically.
func (a *AtomicUint32) GetAndIncrement() uint32 {
    for {
        current := a.Get()
        next := current + 1
        if a.CompareAndSet(current, next) {
            return current
        }
    }

}

// GetAndDecrement gets the old value and then decrement by 1, this operation
// performs atomically.
func (a *AtomicUint32) GetAndDecrement() uint32 {
    for {
        current := a.Get()
        next := current - 1
        if a.CompareAndSet(current, next) {
            return current
        }
    }
}

// GetAndAdd gets the old value and then add by delta, this operation
// performs atomically.
func (a *AtomicUint32) GetAndAdd(delta uint32) uint32 {
    for {
        current := a.Get()
        next := current + delta
        if a.CompareAndSet(current, next) {
            return current
        }
    }
}

// IncrementAndGet increments the value by 1 and then gets the value, this
// operation performs atomically.
func (a *AtomicUint32) IncrementAndGet() uint32 {
    for {
        current := a.Get()
        next := current + 1
        if a.CompareAndSet(current, next) {
            return next
        }
    }
}

// DecrementAndGet decrements the value by 1 and then gets the value, this
// operation performs atomically.
func (a *AtomicUint32) DecrementAndGet() uint32 {
    for {
        current := a.Get()
        next := current - 1
        if a.CompareAndSet(current, next) {
            return next
        }
    }
}

// AddAndGet adds the value by delta and then gets the value, this operation
// performs atomically.
func (a *AtomicUint32) AddAndGet(delta uint32) uint32 {
    for {
        current := a.Get()
        next := current + delta
        if a.CompareAndSet(current, next) {
            return next
        }
    }
}

func (a *AtomicUint32) String() string {
    return fmt.Sprintf("%d", a.Get())
}

// AtomicBoolean provides atomic boolean type.
type AtomicBoolean uint32

// NewAtomicBoolean returns an atomic boolean type.
func NewAtomicBoolean(initialValue bool) *AtomicBoolean {
    var a AtomicBoolean
    if initialValue {
        a = AtomicBoolean(1)
    } else {
        a = AtomicBoolean(0)
    }
    return &a
}

// Get returns the value of boolean atomically.
func (a *AtomicBoolean) Get() bool {
    return atomic.LoadUint32((*uint32)(a)) != 0
}

// Set sets the value of boolean atomically.
func (a *AtomicBoolean) Set(newValue bool) {
    if newValue {
        atomic.StoreUint32((*uint32)(a), 1)
    } else {
        atomic.StoreUint32((*uint32)(a), 0)
    }
}

// CompareAndSet compares boolean with expected value, if equals as expected
// then sets the updated value, this operation performs atomically.
func (a *AtomicBoolean) CompareAndSet(oldValue, newValue bool) bool {
    var o uint32
    var n uint32
    if oldValue {
        o = 1
    } else {
        o = 0
    }
    if newValue {
        n = 1
    } else {
        n = 0
    }
    return atomic.CompareAndSwapUint32((*uint32)(a), o, n)
}

// GetAndSet sets new value and returns the old atomically.
func (a *AtomicBoolean) GetAndSet(newValue bool) bool {
    for {
        current := a.Get()
        if a.CompareAndSet(current, newValue) {
            return current
        }
    }
}

func (a *AtomicBoolean) String() string {
    return fmt.Sprintf("%t", a.Get())
}

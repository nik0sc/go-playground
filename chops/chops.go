// Package chops provides useful channel operations
// that are not provided by the standard `<-` mechanism.
// It is not guaranteed to be compatible with all versions
// of Go, although it is tested on Go 1.18.
package chops

import (
	"runtime"
	"strings"
	"sync/atomic"
	"unsafe"
)

// Status represents the result of a non-blocking channel
// operation. It can be Ok, Closed, or Blocked.
type Status int

func (s Status) String() string {
	switch s {
	case Ok:
		return "Ok"
	case Closed:
		return "Closed"
	case Blocked:
		return "Blocked"
	default:
		return "<invalid chops.Status>"
	}
}

const (
	// The channel accepted the send or receive without
	// blocking.
	Ok Status = iota
	// The channel is closed. Future operations will always
	// return Closed again.
	Closed
	// The channel is not ready to accept the operation.
	// Its buffer could be full, or if it's unbuffered, no
	// goroutine is waiting on the other end.
	Blocked
)

type Result[T any] struct {
	value  T
	status Status
}

// Get returns the result of the channel operation:
// If the return Status is Ok, the receive succeeded and
// the returned T is the channel element.
// If the return Status is Closed, the channel is closed
// and the returned T will be the zero value of T.
// If the return Status is Blocked, the channel is empty
// (but not closed, at the time of the receive) and the
// returned T will be the zero value of T.
func (r Result[T]) Get() (T, Status) {
	return r.value, r.status
}

// Match performs an exhaustive match on the Result.
func (r Result[T]) Match(ok func(T), closed, blocked func()) {
	switch r.status {
	case Ok:
		ok(r.value)
	case Closed:
		closed()
	case Blocked:
		blocked()
	default:
		panic("unhandled case in Match")
	}
}

const closeChMsg = "send on closed channel"
const doubleCloseMsg = "close of closed channel"

// Warning: hackery here! Correct as of 1.18.3
type hchan struct {
	_      uint
	_      uint
	_      unsafe.Pointer
	_      uint16
	closed uint32
}

// TryRecv attempts a non-blocking receive from a channel.
func TryRecv[T any](ch <-chan T) Result[T] {
	select {
	case x, ok := <-ch:
		if ok {
			return Result[T]{
				value:  x,
				status: Ok,
			}
		} else {
			return Result[T]{
				status: Closed,
			}
		}
	default:
		return Result[T]{
			status: Blocked,
		}
	}
}

// TrySend attempts a non-blocking send to a channel.
// If the return Status is Ok, the send succeeded.
// If the return Status is Closed, the channel is closed.
// Future calls to TrySend will continue to return Closed.
// If the return Status is Blocked, the channel is either
// full (if it is buffered) or nobody is listening on the
// other end (if it is unbuffered).
func TrySend[T any](ch chan<- T, x T) (stat Status) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		err, ok := r.(runtime.Error)
		if ok && strings.Contains(err.Error(), closeChMsg) {
			stat = Closed
		} else {
			panic(r)
		}
	}()

	select {
	case ch <- x:
		stat = Ok
	default:
		stat = Blocked
	}

	return
}

// TryClose ensures a channel is closed. It returns true
// if the channel was previously open, or false if the
// channel was already closed at the time of the call.
//
// You should not need to use this function normally, since
// only the sender should close a channel.
func TryClose[T any](ch chan<- T) (ok bool) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		err, ok := r.(runtime.Error)
		if ok && strings.Contains(err.Error(), doubleCloseMsg) {
			ok = false
		} else {
			panic(r)
		}
	}()
	close(ch)
	ok = true
	return
}

// IsClosed returns true if the channel provided is closed.
// You cannot assume that the channel is not closed if this
// function returns false. The channel may still contain
// data to be read, use `len()` to determine that.
func IsClosed[T any](ch chan T) bool {
	c := *(**hchan)(unsafe.Pointer(&ch))
	// See the start of runtime.chanrecv for more details
	return atomic.LoadUint32(&c.closed) != 0
}

// RecvOr, SendOr are pointless with generics

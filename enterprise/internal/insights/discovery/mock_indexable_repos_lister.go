// Code generated by go-mockgen 1.1.5; DO NOT EDIT.

package discovery

import (
	"context"
	"sync"

	types "github.com/sourcegraph/sourcegraph/internal/types"
)

// MockIndexableReposLister is a mock implementation of the
// IndexableReposLister interface (from the package
// github.com/sourcegraph/sourcegraph/enterprise/internal/insights/discovery)
// used for unit testing.
type MockIndexableReposLister struct {
	// ListFunc is an instance of a mock function object controlling the
	// behavior of the method List.
	ListFunc *IndexableReposListerListFunc
}

// NewMockIndexableReposLister creates a new mock of the
// IndexableReposLister interface. All methods return zero values for all
// results, unless overwritten.
func NewMockIndexableReposLister() *MockIndexableReposLister {
	return &MockIndexableReposLister{
		ListFunc: &IndexableReposListerListFunc{
			defaultHook: func(context.Context) ([]types.MinimalRepo, error) {
				return nil, nil
			},
		},
	}
}

// NewStrictMockIndexableReposLister creates a new mock of the
// IndexableReposLister interface. All methods panic on invocation, unless
// overwritten.
func NewStrictMockIndexableReposLister() *MockIndexableReposLister {
	return &MockIndexableReposLister{
		ListFunc: &IndexableReposListerListFunc{
			defaultHook: func(context.Context) ([]types.MinimalRepo, error) {
				panic("unexpected invocation of MockIndexableReposLister.List")
			},
		},
	}
}

// NewMockIndexableReposListerFrom creates a new mock of the
// MockIndexableReposLister interface. All methods delegate to the given
// implementation, unless overwritten.
func NewMockIndexableReposListerFrom(i IndexableReposLister) *MockIndexableReposLister {
	return &MockIndexableReposLister{
		ListFunc: &IndexableReposListerListFunc{
			defaultHook: i.List,
		},
	}
}

// IndexableReposListerListFunc describes the behavior when the List method
// of the parent MockIndexableReposLister instance is invoked.
type IndexableReposListerListFunc struct {
	defaultHook func(context.Context) ([]types.MinimalRepo, error)
	hooks       []func(context.Context) ([]types.MinimalRepo, error)
	history     []IndexableReposListerListFuncCall
	mutex       sync.Mutex
}

// List delegates to the next hook function in the queue and stores the
// parameter and result values of this invocation.
func (m *MockIndexableReposLister) List(v0 context.Context) ([]types.MinimalRepo, error) {
	r0, r1 := m.ListFunc.nextHook()(v0)
	m.ListFunc.appendCall(IndexableReposListerListFuncCall{v0, r0, r1})
	return r0, r1
}

// SetDefaultHook sets function that is called when the List method of the
// parent MockIndexableReposLister instance is invoked and the hook queue is
// empty.
func (f *IndexableReposListerListFunc) SetDefaultHook(hook func(context.Context) ([]types.MinimalRepo, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// List method of the parent MockIndexableReposLister instance invokes the
// hook at the front of the queue and discards it. After the queue is empty,
// the default hook function is invoked for any future action.
func (f *IndexableReposListerListFunc) PushHook(hook func(context.Context) ([]types.MinimalRepo, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultHook with a function that returns the
// given values.
func (f *IndexableReposListerListFunc) SetDefaultReturn(r0 []types.MinimalRepo, r1 error) {
	f.SetDefaultHook(func(context.Context) ([]types.MinimalRepo, error) {
		return r0, r1
	})
}

// PushReturn calls PushHook with a function that returns the given values.
func (f *IndexableReposListerListFunc) PushReturn(r0 []types.MinimalRepo, r1 error) {
	f.PushHook(func(context.Context) ([]types.MinimalRepo, error) {
		return r0, r1
	})
}

func (f *IndexableReposListerListFunc) nextHook() func(context.Context) ([]types.MinimalRepo, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *IndexableReposListerListFunc) appendCall(r0 IndexableReposListerListFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of IndexableReposListerListFuncCall objects
// describing the invocations of this function.
func (f *IndexableReposListerListFunc) History() []IndexableReposListerListFuncCall {
	f.mutex.Lock()
	history := make([]IndexableReposListerListFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// IndexableReposListerListFuncCall is an object that describes an
// invocation of method List on an instance of MockIndexableReposLister.
type IndexableReposListerListFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 []types.MinimalRepo
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c IndexableReposListerListFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c IndexableReposListerListFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1}
}

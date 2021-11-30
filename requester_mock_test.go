package main

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

//go:generate minimock -i lesson1.Requester -o ./requester_mock_test.go -n RequesterMock

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
)

// RequesterMock implements Requester
type RequesterMock struct {
	t minimock.Tester

	funcGet          func(ctx context.Context, url string) (p1 Page, err error)
	inspectFuncGet   func(ctx context.Context, url string)
	afterGetCounter  uint64
	beforeGetCounter uint64
	GetMock          mRequesterMockGet
}

// NewRequesterMock returns a mock for Requester
func NewRequesterMock(t minimock.Tester) *RequesterMock {
	m := &RequesterMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.GetMock = mRequesterMockGet{mock: m}
	m.GetMock.callArgs = []*RequesterMockGetParams{}

	return m
}

type mRequesterMockGet struct {
	mock               *RequesterMock
	defaultExpectation *RequesterMockGetExpectation
	expectations       []*RequesterMockGetExpectation

	callArgs []*RequesterMockGetParams
	mutex    sync.RWMutex
}

// RequesterMockGetExpectation specifies expectation struct of the Requester.Get
type RequesterMockGetExpectation struct {
	mock    *RequesterMock
	params  *RequesterMockGetParams
	results *RequesterMockGetResults
	Counter uint64
}

// RequesterMockGetParams contains parameters of the Requester.Get
type RequesterMockGetParams struct {
	ctx context.Context
	url string
}

// RequesterMockGetResults contains results of the Requester.Get
type RequesterMockGetResults struct {
	p1  Page
	err error
}

// Expect sets up expected params for Requester.Get
func (mmGet *mRequesterMockGet) Expect(ctx context.Context, url string) *mRequesterMockGet {
	if mmGet.mock.funcGet != nil {
		mmGet.mock.t.Fatalf("RequesterMock.Get mock is already set by Set")
	}

	if mmGet.defaultExpectation == nil {
		mmGet.defaultExpectation = &RequesterMockGetExpectation{}
	}

	mmGet.defaultExpectation.params = &RequesterMockGetParams{ctx, url}
	for _, e := range mmGet.expectations {
		if minimock.Equal(e.params, mmGet.defaultExpectation.params) {
			mmGet.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmGet.defaultExpectation.params)
		}
	}

	return mmGet
}

// Inspect accepts an inspector function that has same arguments as the Requester.Get
func (mmGet *mRequesterMockGet) Inspect(f func(ctx context.Context, url string)) *mRequesterMockGet {
	if mmGet.mock.inspectFuncGet != nil {
		mmGet.mock.t.Fatalf("Inspect function is already set for RequesterMock.Get")
	}

	mmGet.mock.inspectFuncGet = f

	return mmGet
}

// Return sets up results that will be returned by Requester.Get
func (mmGet *mRequesterMockGet) Return(p1 Page, err error) *RequesterMock {
	if mmGet.mock.funcGet != nil {
		mmGet.mock.t.Fatalf("RequesterMock.Get mock is already set by Set")
	}

	if mmGet.defaultExpectation == nil {
		mmGet.defaultExpectation = &RequesterMockGetExpectation{mock: mmGet.mock}
	}
	mmGet.defaultExpectation.results = &RequesterMockGetResults{p1, err}
	return mmGet.mock
}

//Set uses given function f to mock the Requester.Get method
func (mmGet *mRequesterMockGet) Set(f func(ctx context.Context, url string) (p1 Page, err error)) *RequesterMock {
	if mmGet.defaultExpectation != nil {
		mmGet.mock.t.Fatalf("Default expectation is already set for the Requester.Get method")
	}

	if len(mmGet.expectations) > 0 {
		mmGet.mock.t.Fatalf("Some expectations are already set for the Requester.Get method")
	}

	mmGet.mock.funcGet = f
	return mmGet.mock
}

// When sets expectation for the Requester.Get which will trigger the result defined by the following
// Then helper
func (mmGet *mRequesterMockGet) When(ctx context.Context, url string) *RequesterMockGetExpectation {
	if mmGet.mock.funcGet != nil {
		mmGet.mock.t.Fatalf("RequesterMock.Get mock is already set by Set")
	}

	expectation := &RequesterMockGetExpectation{
		mock:   mmGet.mock,
		params: &RequesterMockGetParams{ctx, url},
	}
	mmGet.expectations = append(mmGet.expectations, expectation)
	return expectation
}

// Then sets up Requester.Get return parameters for the expectation previously defined by the When method
func (e *RequesterMockGetExpectation) Then(p1 Page, err error) *RequesterMock {
	e.results = &RequesterMockGetResults{p1, err}
	return e.mock
}

// Get implements Requester
func (mmGet *RequesterMock) Get(ctx context.Context, url string) (p1 Page, err error) {
	mm_atomic.AddUint64(&mmGet.beforeGetCounter, 1)
	defer mm_atomic.AddUint64(&mmGet.afterGetCounter, 1)

	if mmGet.inspectFuncGet != nil {
		mmGet.inspectFuncGet(ctx, url)
	}

	mm_params := &RequesterMockGetParams{ctx, url}

	// Record call args
	mmGet.GetMock.mutex.Lock()
	mmGet.GetMock.callArgs = append(mmGet.GetMock.callArgs, mm_params)
	mmGet.GetMock.mutex.Unlock()

	for _, e := range mmGet.GetMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.p1, e.results.err
		}
	}

	if mmGet.GetMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGet.GetMock.defaultExpectation.Counter, 1)
		mm_want := mmGet.GetMock.defaultExpectation.params
		mm_got := RequesterMockGetParams{ctx, url}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmGet.t.Errorf("RequesterMock.Get got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmGet.GetMock.defaultExpectation.results
		if mm_results == nil {
			mmGet.t.Fatal("No results are set for the RequesterMock.Get")
		}
		return (*mm_results).p1, (*mm_results).err
	}
	if mmGet.funcGet != nil {
		return mmGet.funcGet(ctx, url)
	}
	mmGet.t.Fatalf("Unexpected call to RequesterMock.Get. %v %v", ctx, url)
	return
}

// GetAfterCounter returns a count of finished RequesterMock.Get invocations
func (mmGet *RequesterMock) GetAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGet.afterGetCounter)
}

// GetBeforeCounter returns a count of RequesterMock.Get invocations
func (mmGet *RequesterMock) GetBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGet.beforeGetCounter)
}

// Calls returns a list of arguments used in each call to RequesterMock.Get.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmGet *mRequesterMockGet) Calls() []*RequesterMockGetParams {
	mmGet.mutex.RLock()

	argCopy := make([]*RequesterMockGetParams, len(mmGet.callArgs))
	copy(argCopy, mmGet.callArgs)

	mmGet.mutex.RUnlock()

	return argCopy
}

// MinimockGetDone returns true if the count of the Get invocations corresponds
// the number of defined expectations
func (m *RequesterMock) MinimockGetDone() bool {
	for _, e := range m.GetMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGet != nil && mm_atomic.LoadUint64(&m.afterGetCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetInspect logs each unmet expectation
func (m *RequesterMock) MinimockGetInspect() {
	for _, e := range m.GetMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to RequesterMock.Get with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetCounter) < 1 {
		if m.GetMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to RequesterMock.Get")
		} else {
			m.t.Errorf("Expected call to RequesterMock.Get with params: %#v", *m.GetMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGet != nil && mm_atomic.LoadUint64(&m.afterGetCounter) < 1 {
		m.t.Error("Expected call to RequesterMock.Get")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *RequesterMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockGetInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *RequesterMock) MinimockWait(timeout mm_time.Duration) {
	timeoutCh := mm_time.After(timeout)
	for {
		if m.minimockDone() {
			return
		}
		select {
		case <-timeoutCh:
			m.MinimockFinish()
			return
		case <-mm_time.After(10 * mm_time.Millisecond):
		}
	}
}

func (m *RequesterMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockGetDone()
}

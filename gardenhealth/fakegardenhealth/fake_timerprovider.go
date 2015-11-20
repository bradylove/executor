// This file was generated by counterfeiter
package fakegardenhealth

import (
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/executor/gardenhealth"
	"github.com/pivotal-golang/clock"
)

type FakeTimerProvider struct {
	NewTimerStub        func(time.Duration) clock.Timer
	newTimerMutex       sync.RWMutex
	newTimerArgsForCall []struct {
		arg1 time.Duration
	}
	newTimerReturns struct {
		result1 clock.Timer
	}
}

func (fake *FakeTimerProvider) NewTimer(arg1 time.Duration) clock.Timer {
	fake.newTimerMutex.Lock()
	fake.newTimerArgsForCall = append(fake.newTimerArgsForCall, struct {
		arg1 time.Duration
	}{arg1})
	fake.newTimerMutex.Unlock()
	if fake.NewTimerStub != nil {
		return fake.NewTimerStub(arg1)
	} else {
		return fake.newTimerReturns.result1
	}
}

func (fake *FakeTimerProvider) NewTimerCallCount() int {
	fake.newTimerMutex.RLock()
	defer fake.newTimerMutex.RUnlock()
	return len(fake.newTimerArgsForCall)
}

func (fake *FakeTimerProvider) NewTimerArgsForCall(i int) time.Duration {
	fake.newTimerMutex.RLock()
	defer fake.newTimerMutex.RUnlock()
	return fake.newTimerArgsForCall[i].arg1
}

func (fake *FakeTimerProvider) NewTimerReturns(result1 clock.Timer) {
	fake.newTimerMutex.Lock()
	defer fake.newTimerMutex.Unlock()
	fake.NewTimerStub = nil
	fake.newTimerReturns = struct {
		result1 clock.Timer
	}{result1}
}

var _ gardenhealth.TimerProvider = new(FakeTimerProvider)
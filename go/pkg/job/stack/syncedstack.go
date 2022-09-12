package stack

import (
	"sync"
)

type SyncedStack struct {
	Stack  Stacker
	block  bool
	mutex  *sync.RWMutex
	signal chan chan bool
}

func NewSyncedStack(stack Stacker, block bool) *SyncedStack {
	return &SyncedStack{
		Stack:  stack,
		block:  block,
		mutex:  &sync.RWMutex{},
		signal: make(chan chan bool, 1000),
	}
}

func (cs *SyncedStack) Push(v interface{}) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	cs.Stack.Push(v)
	select {
	case done := <-cs.signal:
		done <- true
	default:
	}
}

func (cs *SyncedStack) Block() {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	cs.block = true
}

func (cs *SyncedStack) Unblock() {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	defer func() { cs.block = false }()
	//log.Printf("Trying to unblock")
	for {
		select {
		case done := <-cs.signal:
			done <- false
		default:
			//log.Printf("Nothing to unblock")
			return
		}
	}
}

func (cs *SyncedStack) Pop() (interface{}, bool) {
	for {
		cs.mutex.Lock()
		if v, ok := cs.Stack.Pop(); ok || !cs.block {
			cs.mutex.Unlock()
			return v, ok
		}
		cs.mutex.Unlock()
		wait := make(chan bool)
		cs.signal <- wait
		if !<-wait {
			return nil, false
		}
	}
}

func (cs *SyncedStack) Len() int {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	return cs.Stack.Len()
}

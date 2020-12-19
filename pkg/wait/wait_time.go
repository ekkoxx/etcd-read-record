// Copyright 2015 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wait

import "sync"

//这个里面就是放 已经apply的index的地方。
//Trigger是在applyAll的时候触发的。
//Wait是需要等待给定的index必须被apply的地方使用的
//   如果Wait的index已经被apply了，会直接返回个已经close的chan struct{}就可以直接用
//   否则将index放进map中，等apply的时候触发trigger，然后index对应的 chan struct{} 会被close 掉
type WaitTime interface {
	// Wait returns a chan that waits on the given logical deadline.
	// The chan will be triggered when Trigger is called with a
	// deadline that is later than the one it is waiting for.
	Wait(deadline uint64) <-chan struct{}
	// Trigger triggers all the waiting chans with an earlier logical deadline.
	Trigger(deadline uint64)
}

var closec chan struct{}

func init() { closec = make(chan struct{}); close(closec) }

type timeList struct {
	l                   sync.Mutex
	lastTriggerDeadline uint64
	m                   map[uint64]chan struct{}
}

func NewTimeList() *timeList {
	return &timeList{m: make(map[uint64]chan struct{})}
}

func (tl *timeList) Wait(deadline uint64) <-chan struct{} {
	tl.l.Lock()
	defer tl.l.Unlock()
	if tl.lastTriggerDeadline >= deadline {
		return closec
	}
	ch := tl.m[deadline]
	if ch == nil {
		ch = make(chan struct{})
		tl.m[deadline] = ch
	}
	return ch
}

func (tl *timeList) Trigger(deadline uint64) {
	tl.l.Lock()
	defer tl.l.Unlock()
	tl.lastTriggerDeadline = deadline
	for t, ch := range tl.m {
		if t <= deadline {
			delete(tl.m, t)
			close(ch)
		}
	}
}

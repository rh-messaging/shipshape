/*
Copyright 2019 The Interconnectedcloud Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import "sync"

// CleanupActionHandle is an integer pointer type for handling cleanup action
type CleanupActionHandle *int

// Action type to be enqueued
type ActionType int

const (
	AfterEach ActionType = iota
	AfterSuite
)

var cleanupActionsLock sync.Mutex
var cleanupActionsEach = map[CleanupActionHandle]func(){}
var cleanupActionsSuite = map[CleanupActionHandle]func(){}

// AddCleanupAction installs a function that will be called in the event of
// completion of a test Spec or a test Suite.  This allows arbitrary pieces of the overall
// test to hook into AfterEach() and SynchronizedAfterSuite().
func AddCleanupAction(action ActionType, fn func()) CleanupActionHandle {
	p := CleanupActionHandle(new(int))
	cleanupActionsLock.Lock()
	defer cleanupActionsLock.Unlock()
	switch action {
	case AfterEach:
		cleanupActionsEach[p] = fn
	case AfterSuite:
		cleanupActionsSuite[p] = fn
	default:
		cleanupActionsEach[p] = fn
	}
	return p
}

// RemoveCleanupAction removes a function that was installed by
// AddCleanupAction.
func RemoveCleanupAction(action ActionType, p CleanupActionHandle) {
	cleanupActionsLock.Lock()
	defer cleanupActionsLock.Unlock()
	switch action {
	case AfterEach:
		delete(cleanupActionsEach, p)
	case AfterSuite:
		delete(cleanupActionsSuite, p)
	default:
		delete(cleanupActionsEach, p)
	}
}

// RunCleanupActions runs all functions installed by AddCleanupAction.  It does
// not remove them (see RemoveCleanupAction) but it does run unlocked, so they
// may remove themselves.
func RunCleanupActions(action ActionType) {
	list := []func(){}
	func() {
		cleanupActionsLock.Lock()
		defer cleanupActionsLock.Unlock()
		for _, fn := range cleanupActionsEach {
			list = append(list, fn)
		}
	}()
	// Run unlocked.
	for _, fn := range list {
		fn()
	}
}

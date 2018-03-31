// Copyright 2018 The Jadep Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package future implements future/promise primitives.
package future

// Value implements a future/promise for an arbitrary value.
type Value struct {
	value interface{}

	// ready is a broadcast channel
	ready chan bool
}

// NewValue returns a new Value future, whose value is computed by f().
// f() is called concurrently - NewValue doesn't block.
func NewValue(f func() interface{}) *Value {
	result := &Value{nil, make(chan bool)}
	go func() {
		result.value = f()
		close(result.ready)
	}()
	return result
}

// Get returns the value computed by the function given to NewValue.
// It blocks until the value is ready.
func (f *Value) Get() interface{} {
	<-f.ready
	return f.value
}

// Immediate returns a Value which resolves to 'value'.
func Immediate(value interface{}) *Value {
	return NewValue(func() interface{} { return value })
}

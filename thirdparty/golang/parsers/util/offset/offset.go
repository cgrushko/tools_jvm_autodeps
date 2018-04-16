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

// Package offset provides a helper converting byte offsets in a string into (line, column) pairs,
// where 'column' represents the number of runes from the beginning of the line.
package offset

import (
	"fmt"
	"sort"
	"unicode/utf8"
)

var (
	errInvalidLine   = fmt.Errorf("invalid line number")
	errInvalidColumn = fmt.Errorf("invalid column number")
	errInvalidOffset = fmt.Errorf("invalid offset value")
)

// Range contains the start and end byte offsets of a range of a string (both 0-based).
type Range struct {
	Offset    int
	EndOffset int
}

// Empty reports whether r is empty.
func (r Range) Empty() bool {
	return r.Offset == r.EndOffset
}

// Contains checks if the offset belongs to the range (inclusive on both sides).
func (r Range) Contains(offset int) bool {
	return r.Offset <= offset && offset <= r.EndOffset
}

// ContainsRange checks if the given range is fully contained in "this" range.
func (r Range) ContainsRange(or Range) bool {
	return r.Offset <= or.Offset && or.EndOffset <= r.EndOffset
}

func (r Range) String() string {
	return fmt.Sprintf("(%v, %v)", r.Offset, r.EndOffset)
}

// Mapper converts file locations between byte offset and line-and-column representations.
// Lines and columns are 0-based and represent characters (runes), while offsets refer to byte offsets.
type Mapper struct {
	offsets []int // byte offsets for beginning of lines
	content string
}

// NewMapper returns an offset mapper for the content provided as input.
func NewMapper(content string) *Mapper {
	m := &Mapper{
		content: content,
	}

	// Precompute line offsets, for easy deduction of the line number for a global offset
	m.offsets = make([]int, 0, 32)
	m.offsets = append(m.offsets, 0) // First line starts at offset 0.
	for offset, r := range content {
		if r == '\n' {
			m.offsets = append(m.offsets, offset+1)
		}
	}

	// Introduce an artificial last line.
	m.offsets = append(m.offsets, len(content))

	return m
}

// ByteOffset returns the global byte offset equivalent to the (line, column) values which represent runes.
// Line and column are 0-based values.
func (m *Mapper) ByteOffset(line, column int) (int, error) {
	if line < 0 || line >= len(m.offsets)-1 {
		return 0, errInvalidLine
	}
	if column < 0 {
		return 0, errInvalidColumn
	}
	if column == 0 {
		return m.offsets[line], nil
	}

	lineText := m.content[m.offsets[line]:m.offsets[line+1]]

	columnIndex := 0
	for offset := range lineText {
		columnIndex++
		if columnIndex == column+1 {
			return m.offsets[line] + offset, nil
		}
	}
	if line == len(m.offsets)-2 && utf8.RuneCountInString(lineText) == column {
		// The line and column point to the end of file.
		return len(m.content), nil
	}

	return 0, errInvalidColumn
}

// LineAndColumn returns line and column values (rune-based) from a global byte offset.
// If the offset is pointing a byte in the middle of a rune, then it returns the column right after the rune.
// Both the line and column are 0-based.
// Returns an error if the input offset is negative or greater than the length of the source content.
func (m *Mapper) LineAndColumn(byteOffset int) (line, column int, err error) {
	if byteOffset > len(m.content) || byteOffset < 0 {
		err = errInvalidOffset
		return
	}
	if byteOffset == len(m.content) {
		// Offset points to end of file.
		line = len(m.offsets) - 2
		column = utf8.RuneCountInString(m.content[m.offsets[line]:m.offsets[line+1]])
		return
	}

	line = sort.Search(len(m.offsets), func(i int) bool {
		return m.offsets[i] > byteOffset
	}) - 1

	lineStartOffset := m.offsets[line]

	lineText := m.content[m.offsets[line]:m.offsets[line+1]]

	column = -1
	for lineOffset := range lineText {
		column++
		if lineOffset+lineStartOffset >= byteOffset {
			return
		}
	}
	return
}

// LineOffsets returns a list with the starting file offsets for each line.
// The first element is 0, while the last element is len(content).
func (m *Mapper) LineOffsets() []int {
	// Remove the last, artificial line.
	return m.offsets[:len(m.offsets)-1]
}

// NumLines returns the number of lines in the input.
func (m *Mapper) NumLines() int {
	return len(m.offsets) - 1
}

// Len returns the length of the input.
func (m *Mapper) Len() int {
	return len(m.content)
}

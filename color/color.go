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

// Package color colorizes output sent to a terminal
package color

const keyEscape = 27

var (
	// Enabled decides whether the colorization functions (e.g. Green()) are no-ops.
	Enabled = true

	black    = []byte{keyEscape, '[', '3', '0', 'm'}
	red      = []byte{keyEscape, '[', '3', '1', 'm'}
	green    = []byte{keyEscape, '[', '3', '2', 'm'}
	yellow   = []byte{keyEscape, '[', '3', '3', 'm'}
	blue     = []byte{keyEscape, '[', '3', '4', 'm'}
	magenta  = []byte{keyEscape, '[', '3', '5', 'm'}
	cyan     = []byte{keyEscape, '[', '3', '6', 'm'}
	white    = []byte{keyEscape, '[', '3', '7', 'm'}
	darkGray = []byte{keyEscape, '[', '9', '0', 'm'}

	bold = []byte{keyEscape, '[', '1', 'm'}

	reset = []byte{keyEscape, '[', '0', 'm'}
)

func wrap(s string, codes []byte) string {
	if !Enabled {
		return s
	}
	return string(codes) + s + string(reset)
}

// Bold returns s wrapped in ANSI codes which cause terminals to display it bold.
func Bold(s string) string {
	return wrap(s, bold)
}

// Green returns s wrapped in ANSI codes which cause terminals to display it green.
func Green(s string) string {
	return wrap(s, green)
}

// Magenta returns s wrapped in ANSI codes which cause terminals to display it magenta.
func Magenta(s string) string {
	return wrap(s, magenta)
}

// BoldMagenta returns s wrapped in ANSI codes which cause terminals to display it bold magenta.
func BoldMagenta(s string) string {
	return wrap(wrap(s, magenta), bold)
}

// BoldGreen returns s wrapped in ANSI codes which cause terminals to display it bold green.
func BoldGreen(s string) string {
	return wrap(wrap(s, green), bold)
}

// DarkGray returns s wrapped in ANSI codes which cause terminals to display it dark gray.
func DarkGray(s string) string {
	return wrap(s, darkGray)
}

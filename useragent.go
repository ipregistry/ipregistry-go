// Copyright 2019 Ipregistry (https://ipregistry.co).
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

package ipregistry

import "strings"

// UserAgent holds the structured data parsed from a raw User-Agent string.
type UserAgent struct {
	// Header is the raw User-Agent string that was parsed.
	Header          string                   `json:"header,omitempty"`
	Name            string                   `json:"name,omitempty"`
	Type            string                   `json:"type,omitempty"`
	Version         string                   `json:"version,omitempty"`
	VersionMajor    string                   `json:"version_major,omitempty"`
	Device          UserAgentDevice          `json:"device"`
	Engine          UserAgentEngine          `json:"engine"`
	OperatingSystem UserAgentOperatingSystem `json:"os"`
}

// UserAgentDevice holds the device data parsed from a User-Agent string.
type UserAgentDevice struct {
	Brand string `json:"brand,omitempty"`
	Name  string `json:"name,omitempty"`
	Type  string `json:"type,omitempty"`
}

// UserAgentEngine holds the layout-engine data parsed from a User-Agent string.
type UserAgentEngine struct {
	Name         string `json:"name,omitempty"`
	Type         string `json:"type,omitempty"`
	Version      string `json:"version,omitempty"`
	VersionMajor string `json:"version_major,omitempty"`
}

// UserAgentOperatingSystem holds the OS data parsed from a User-Agent string.
type UserAgentOperatingSystem struct {
	Name    string `json:"name,omitempty"`
	Type    string `json:"type,omitempty"`
	Version string `json:"version,omitempty"`
}

// IsBot reports whether the given raw User-Agent string looks like a crawler or
// bot. It is a lightweight heuristic — useful for skipping IP lookups on
// automated traffic — that matches the substrings "bot", "spider", and "slurp"
// case-insensitively.
func IsBot(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	return strings.Contains(ua, "bot") ||
		strings.Contains(ua, "spider") ||
		strings.Contains(ua, "slurp")
}

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

import (
	"encoding/json"
	"iter"
)

// IPInfoResult is one entry of a batch lookup. Exactly one of Info and Err is
// non-nil: Info holds the data for a successfully resolved IP address, and Err
// describes why that particular entry failed.
type IPInfoResult struct {
	Info *IPInfo
	Err  *APIError
}

// IPInfoList holds the ordered results of a batch IP lookup. Each entry may
// independently succeed or fail; use At or All to inspect them.
type IPInfoList struct {
	Results []IPInfoResult
}

// Len returns the number of entries in the list.
func (l *IPInfoList) Len() int {
	return len(l.Results)
}

// At returns the IPInfo at index i, or the error that entry failed with. It
// panics if i is out of range, matching slice indexing semantics.
func (l *IPInfoList) At(i int) (*IPInfo, error) {
	r := l.Results[i]
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Info, nil
}

// All returns an iterator over the entries in order, yielding either the
// resolved IPInfo or the error for each entry. It is intended for use with
// Go's range-over-func:
//
//	for info, err := range list.All() {
//		if err != nil {
//			// handle this entry's failure
//			continue
//		}
//		// use info
//	}
func (l *IPInfoList) All() iter.Seq2[*IPInfo, error] {
	return func(yield func(*IPInfo, error) bool) {
		for _, r := range l.Results {
			if r.Err != nil {
				if !yield(nil, r.Err) {
					return
				}
			} else if !yield(r.Info, nil) {
				return
			}
		}
	}
}

// UnmarshalJSON decodes the {"results": [...]} envelope, mapping each element
// to either an IPInfo or an APIError depending on whether it carries an error
// code.
func (l *IPInfoList) UnmarshalJSON(data []byte) error {
	envelope, err := decodeResults(data)
	if err != nil {
		return err
	}

	l.Results = make([]IPInfoResult, len(envelope))
	for i, raw := range envelope {
		if isErrorEntry(raw) {
			var p apiErrorPayload
			if err := json.Unmarshal(raw, &p); err != nil {
				return err
			}
			l.Results[i] = IPInfoResult{Err: p.toAPIError()}
			continue
		}
		info := new(IPInfo)
		if err := json.Unmarshal(raw, info); err != nil {
			return err
		}
		l.Results[i] = IPInfoResult{Info: info}
	}
	return nil
}

// UserAgentResult is one entry of a batch User-Agent parse. Exactly one of
// UserAgent and Err is non-nil.
type UserAgentResult struct {
	UserAgent *UserAgent
	Err       *APIError
}

// UserAgentList holds the ordered results of a User-Agent parse request.
type UserAgentList struct {
	Results []UserAgentResult
}

// Len returns the number of entries in the list.
func (l *UserAgentList) Len() int {
	return len(l.Results)
}

// At returns the UserAgent at index i, or the error that entry failed with. It
// panics if i is out of range, matching slice indexing semantics.
func (l *UserAgentList) At(i int) (*UserAgent, error) {
	r := l.Results[i]
	if r.Err != nil {
		return nil, r.Err
	}
	return r.UserAgent, nil
}

// All returns an iterator over the entries in order, yielding either the parsed
// UserAgent or the error for each entry.
func (l *UserAgentList) All() iter.Seq2[*UserAgent, error] {
	return func(yield func(*UserAgent, error) bool) {
		for _, r := range l.Results {
			if r.Err != nil {
				if !yield(nil, r.Err) {
					return
				}
			} else if !yield(r.UserAgent, nil) {
				return
			}
		}
	}
}

// UnmarshalJSON decodes the {"results": [...]} envelope, mapping each element
// to either a UserAgent or an APIError depending on whether it carries an error
// code.
func (l *UserAgentList) UnmarshalJSON(data []byte) error {
	envelope, err := decodeResults(data)
	if err != nil {
		return err
	}

	l.Results = make([]UserAgentResult, len(envelope))
	for i, raw := range envelope {
		if isErrorEntry(raw) {
			var p apiErrorPayload
			if err := json.Unmarshal(raw, &p); err != nil {
				return err
			}
			l.Results[i] = UserAgentResult{Err: p.toAPIError()}
			continue
		}
		ua := new(UserAgent)
		if err := json.Unmarshal(raw, ua); err != nil {
			return err
		}
		l.Results[i] = UserAgentResult{UserAgent: ua}
	}
	return nil
}

// decodeResults extracts the raw JSON elements of the "results" array.
func decodeResults(data []byte) ([]json.RawMessage, error) {
	var envelope struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	return envelope.Results, nil
}

// isErrorEntry reports whether a results element is an error object, detected by
// the presence of a non-null "code" field.
func isErrorEntry(raw json.RawMessage) bool {
	var probe struct {
		Code *string `json:"code"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return false
	}
	return probe.Code != nil
}

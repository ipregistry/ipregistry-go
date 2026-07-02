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
	"net/url"
	"strconv"
)

// LookupOption customizes a single lookup or batch request by setting a query
// parameter. Construct options with WithFields, WithHostname, or WithParam.
type LookupOption interface {
	apply(url.Values)
}

// lookupOptionFunc adapts a function to the LookupOption interface.
type lookupOptionFunc func(url.Values)

func (f lookupOptionFunc) apply(v url.Values) { f(v) }

// WithFields restricts the response to the given fields, using Ipregistry's
// field selector syntax (for example "location.country.name,security"). This
// reduces payload size and, in some cases, credit usage. See
// https://ipregistry.co/docs/filtering-selecting-fields for the syntax.
func WithFields(expression string) LookupOption {
	return lookupOptionFunc(func(v url.Values) {
		v.Set("fields", expression)
	})
}

// WithHostname enables or disables reverse-DNS hostname resolution for the
// looked-up IP addresses. It is disabled by default.
func WithHostname(enabled bool) LookupOption {
	return lookupOptionFunc(func(v url.Values) {
		v.Set("hostname", strconv.FormatBool(enabled))
	})
}

// WithParam sets an arbitrary query parameter. Use it for options not covered
// by a dedicated helper.
func WithParam(name, value string) LookupOption {
	return lookupOptionFunc(func(v url.Values) {
		v.Set(name, value)
	})
}

// buildParams collapses lookup options into a url.Values.
func buildParams(opts []LookupOption) url.Values {
	v := url.Values{}
	for _, o := range opts {
		o.apply(v)
	}
	return v
}

// cacheKey derives a deterministic cache key from an IP address and its query
// parameters. url.Values.Encode sorts keys, so the key is stable regardless of
// option ordering.
func cacheKey(ip string, params url.Values) string {
	if len(params) == 0 {
		return ip
	}
	return ip + ";" + params.Encode()
}

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
	"fmt"
	"strings"
)

// ErrorCode is a strongly typed Ipregistry API error code. It lets callers
// branch on error conditions without matching on raw strings. See
// https://ipregistry.co/docs/errors for the authoritative list.
type ErrorCode string

const (
	ErrorCodeBadRequest          ErrorCode = "BAD_REQUEST"
	ErrorCodeDisabledAPIKey      ErrorCode = "DISABLED_API_KEY"
	ErrorCodeForbiddenIP         ErrorCode = "FORBIDDEN_IP"
	ErrorCodeForbiddenOrigin     ErrorCode = "FORBIDDEN_ORIGIN"
	ErrorCodeForbiddenIPOrigin   ErrorCode = "FORBIDDEN_IP_ORIGIN"
	ErrorCodeInternal            ErrorCode = "INTERNAL"
	ErrorCodeInsufficientCredits ErrorCode = "INSUFFICIENT_CREDITS"
	ErrorCodeInvalidAPIKey       ErrorCode = "INVALID_API_KEY"
	ErrorCodeInvalidASN          ErrorCode = "INVALID_ASN"
	ErrorCodeInvalidFilterSyntax ErrorCode = "INVALID_FILTER_SYNTAX"
	ErrorCodeInvalidIPAddress    ErrorCode = "INVALID_IP_ADDRESS"
	ErrorCodeMissingAPIKey       ErrorCode = "MISSING_API_KEY"
	ErrorCodeReservedASN         ErrorCode = "RESERVED_ASN"
	ErrorCodeReservedIPAddress   ErrorCode = "RESERVED_IP_ADDRESS"
	ErrorCodeTooManyASNs         ErrorCode = "TOO_MANY_ASNS"
	ErrorCodeTooManyIPs          ErrorCode = "TOO_MANY_IPS"
	ErrorCodeTooManyRequests     ErrorCode = "TOO_MANY_REQUESTS"
	ErrorCodeTooManyUserAgents   ErrorCode = "TOO_MANY_USER_AGENTS"
	ErrorCodeUnknownASN          ErrorCode = "UNKNOWN_ASN"
)

// knownErrorCodes is the set of codes ParseErrorCode recognizes.
var knownErrorCodes = map[ErrorCode]struct{}{
	ErrorCodeBadRequest:          {},
	ErrorCodeDisabledAPIKey:      {},
	ErrorCodeForbiddenIP:         {},
	ErrorCodeForbiddenOrigin:     {},
	ErrorCodeForbiddenIPOrigin:   {},
	ErrorCodeInternal:            {},
	ErrorCodeInsufficientCredits: {},
	ErrorCodeInvalidAPIKey:       {},
	ErrorCodeInvalidASN:          {},
	ErrorCodeInvalidFilterSyntax: {},
	ErrorCodeInvalidIPAddress:    {},
	ErrorCodeMissingAPIKey:       {},
	ErrorCodeReservedASN:         {},
	ErrorCodeReservedIPAddress:   {},
	ErrorCodeTooManyASNs:         {},
	ErrorCodeTooManyIPs:          {},
	ErrorCodeTooManyRequests:     {},
	ErrorCodeTooManyUserAgents:   {},
	ErrorCodeUnknownASN:          {},
}

// ParseErrorCode maps a raw API error code to its typed [ErrorCode]. It returns
// an empty ErrorCode when the raw code is not recognized.
func ParseErrorCode(raw string) ErrorCode {
	code := ErrorCode(strings.ToUpper(strings.TrimSpace(raw)))
	if _, ok := knownErrorCodes[code]; ok {
		return code
	}
	return ""
}

// APIError is returned when the Ipregistry API reports a failure, such as an
// invalid IP address, an exhausted credit balance, or throttling. It carries
// both the raw Code and, when recognized, the typed ErrorCode.
//
// In batch lookups, an APIError may also describe the failure of a single entry
// rather than the whole request (see IPInfoList and UserAgentList).
type APIError struct {
	// Code is the raw error code returned by the API.
	Code string
	// ErrorCode is the typed form of Code, or empty if Code is not recognized.
	ErrorCode ErrorCode
	// Message is a human-readable description of the error.
	Message string
	// Resolution suggests how to resolve the error, when available.
	Resolution string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	var b strings.Builder
	b.WriteString("ipregistry: ")
	if e.Message != "" {
		b.WriteString(e.Message)
	} else {
		b.WriteString("API error")
	}
	if e.Code != "" {
		fmt.Fprintf(&b, " (%s)", e.Code)
	}
	if e.Resolution != "" {
		fmt.Fprintf(&b, ": %s", e.Resolution)
	}
	return b.String()
}

// ClientError is returned for failures that occur on the client side rather
// than being reported by the API, such as network errors, request
// cancellation, or a response that cannot be decoded. The underlying cause,
// when any, is available through errors.Unwrap.
type ClientError struct {
	Message string
	Err     error
}

// Error implements the error interface.
func (e *ClientError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("ipregistry: %s: %v", e.Message, e.Err)
	}
	return "ipregistry: " + e.Message
}

// Unwrap returns the underlying cause, enabling errors.Is and errors.As.
func (e *ClientError) Unwrap() error {
	return e.Err
}

// apiErrorPayload mirrors the JSON error body returned by the API.
type apiErrorPayload struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Resolution string `json:"resolution"`
	IP         string `json:"ip,omitempty"`
}

// toAPIError converts a decoded payload into a typed *APIError.
func (p apiErrorPayload) toAPIError() *APIError {
	return &APIError{
		Code:       p.Code,
		ErrorCode:  ParseErrorCode(p.Code),
		Message:    p.Message,
		Resolution: p.Resolution,
	}
}

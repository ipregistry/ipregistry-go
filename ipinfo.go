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

// IPType denotes the version of an IP address.
type IPType string

const (
	IPTypeIPv4    IPType = "IPv4"
	IPTypeIPv6    IPType = "IPv6"
	IPTypeUnknown IPType = "Unknown"
)

// IPInfo holds the comprehensive set of information associated with an IP
// address returned by the Ipregistry API.
//
// Nested objects (Carrier, Company, Connection, Currency, Location, Security,
// and TimeZone) are always present as values, so accessing their fields never
// panics even when the API omitted them; absent fields hold their zero value.
type IPInfo struct {
	// IP is the IP address the data refers to.
	IP string `json:"ip,omitempty"`
	// Type is the IP version (IPv4, IPv6, or Unknown).
	Type IPType `json:"type,omitempty"`
	// Hostname is the reverse-DNS hostname, when hostname resolution is
	// requested (see WithHostname) and available.
	Hostname string `json:"hostname,omitempty"`

	Carrier    Carrier    `json:"carrier"`
	Company    Company    `json:"company"`
	Connection Connection `json:"connection"`
	Currency   Currency   `json:"currency"`
	Location   Location   `json:"location"`
	Security   Security   `json:"security"`
	TimeZone   TimeZone   `json:"time_zone"`
}

// RequesterIPInfo enriches IPInfo with parsed User-Agent data. It is returned
// by Client.LookupOrigin, where the User-Agent of the calling client is known.
type RequesterIPInfo struct {
	IPInfo
	// UserAgent holds the parsed User-Agent of the requester, or nil when the
	// API did not return any.
	UserAgent *UserAgent `json:"user_agent,omitempty"`
}

// Carrier holds mobile carrier information associated with an IP address.
type Carrier struct {
	Name string `json:"name,omitempty"`
	// MCC is the Mobile Country Code.
	MCC string `json:"mcc,omitempty"`
	// MNC is the Mobile Network Code.
	MNC string `json:"mnc,omitempty"`
}

// CompanyType classifies the kind of company that owns an IP address.
type CompanyType string

const (
	CompanyTypeBusiness   CompanyType = "business"
	CompanyTypeEducation  CompanyType = "education"
	CompanyTypeGovernment CompanyType = "government"
	CompanyTypeHosting    CompanyType = "hosting"
	CompanyTypeISP        CompanyType = "isp"
)

// Company holds ownership information for the IP address.
type Company struct {
	Name   string      `json:"name,omitempty"`
	Domain string      `json:"domain,omitempty"`
	Type   CompanyType `json:"type,omitempty"`
}

// ConnectionType classifies the kind of network the IP address belongs to.
type ConnectionType string

const (
	ConnectionTypeBusiness   ConnectionType = "business"
	ConnectionTypeEducation  ConnectionType = "education"
	ConnectionTypeGovernment ConnectionType = "government"
	ConnectionTypeHosting    ConnectionType = "hosting"
	ConnectionTypeInactive   ConnectionType = "inactive"
	ConnectionTypeISP        ConnectionType = "isp"
)

// Connection holds network connection information for the IP address.
type Connection struct {
	// ASN is the Autonomous System Number, or nil when unknown.
	ASN          *int64         `json:"asn,omitempty"`
	Domain       string         `json:"domain,omitempty"`
	Organization string         `json:"organization,omitempty"`
	Route        string         `json:"route,omitempty"`
	Type         ConnectionType `json:"type,omitempty"`
}

// Currency holds currency information for the IP address location.
type Currency struct {
	Code         string         `json:"code,omitempty"`
	Name         string         `json:"name,omitempty"`
	NameNative   string         `json:"name_native,omitempty"`
	Plural       string         `json:"plural,omitempty"`
	PluralNative string         `json:"plural_native,omitempty"`
	Symbol       string         `json:"symbol,omitempty"`
	SymbolNative string         `json:"symbol_native,omitempty"`
	Format       CurrencyFormat `json:"format"`
}

// CurrencyFormat describes how monetary values are formatted for a currency.
type CurrencyFormat struct {
	DecimalSeparator string              `json:"decimal_separator,omitempty"`
	GroupSeparator   string              `json:"group_separator,omitempty"`
	Negative         CurrencyFormatAffix `json:"negative"`
	Positive         CurrencyFormatAffix `json:"positive"`
}

// CurrencyFormatAffix holds the prefix and suffix applied around a formatted
// monetary value (for example the currency symbol and a sign).
type CurrencyFormatAffix struct {
	Prefix string `json:"prefix"`
	Suffix string `json:"suffix"`
}

// Security holds threat-intelligence flags for the IP address.
type Security struct {
	IsAbuser        bool `json:"is_abuser"`
	IsAttacker      bool `json:"is_attacker"`
	IsBogon         bool `json:"is_bogon"`
	IsCloudProvider bool `json:"is_cloud_provider"`
	IsProxy         bool `json:"is_proxy"`
	IsRelay         bool `json:"is_relay"`
	IsTor           bool `json:"is_tor"`
	IsTorExit       bool `json:"is_tor_exit"`
	IsAnonymous     bool `json:"is_anonymous"`
	IsThreat        bool `json:"is_threat"`
	IsVPN           bool `json:"is_vpn"`
}

// TimeZone holds time zone information for the IP address location.
type TimeZone struct {
	ID           string `json:"id,omitempty"`
	Abbreviation string `json:"abbreviation,omitempty"`
	CurrentTime  string `json:"current_time,omitempty"`
	Name         string `json:"name,omitempty"`
	// Offset is the current offset from UTC in seconds.
	Offset           int  `json:"offset"`
	InDaylightSaving bool `json:"in_daylight_saving"`
}

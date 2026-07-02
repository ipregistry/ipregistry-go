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

// Location holds the geographical location associated with an IP address.
type Location struct {
	Continent Continent `json:"continent"`
	Country   Country   `json:"country"`
	Region    Region    `json:"region"`
	City      string    `json:"city,omitempty"`
	Postal    string    `json:"postal,omitempty"`
	// Latitude is the decimal-degree latitude, or nil when unavailable.
	Latitude *float64 `json:"latitude,omitempty"`
	// Longitude is the decimal-degree longitude, or nil when unavailable.
	Longitude *float64 `json:"longitude,omitempty"`
	// Language is the primary language spoken at the location.
	Language Language `json:"language"`
	// InEU reports whether the location is within a European Union member state.
	InEU bool `json:"in_eu"`
}

// Continent holds continent-level information for a location.
type Continent struct {
	Code string `json:"code,omitempty"`
	Name string `json:"name,omitempty"`
}

// Country holds country-level information for a location.
type Country struct {
	// Area is the total land area in square kilometers.
	Area float64 `json:"area"`
	// Borders lists the ISO 3166-1 alpha-2 codes of bordering countries.
	Borders     []string `json:"borders,omitempty"`
	CallingCode string   `json:"calling_code,omitempty"`
	Capital     string   `json:"capital,omitempty"`
	// Code is the ISO 3166-1 alpha-2 country code (for example "US").
	Code string `json:"code,omitempty"`
	Name string `json:"name,omitempty"`
	// Population is the estimated number of inhabitants.
	Population int `json:"population"`
	// PopulationDensity is the number of inhabitants per square kilometer.
	PopulationDensity float64    `json:"population_density"`
	Flag              Flag       `json:"flag"`
	Languages         []Language `json:"languages,omitempty"`
	// TLD is the country-code top-level domain (for example ".us").
	TLD string `json:"tld,omitempty"`
}

// Region holds administrative region (state/province) information.
type Region struct {
	// Code is typically the ISO 3166-2 subdivision code.
	Code string `json:"code,omitempty"`
	Name string `json:"name,omitempty"`
}

// Language holds language information.
type Language struct {
	Code string `json:"code,omitempty"`
	Name string `json:"name,omitempty"`
	// NativeName is the language's name in the language itself.
	NativeName string `json:"native,omitempty"`
}

// Flag holds representations of a country flag across several icon sets.
type Flag struct {
	Emoji        string `json:"emoji,omitempty"`
	EmojiUnicode string `json:"emoji_unicode,omitempty"`
	Emojitwo     string `json:"emojitwo,omitempty"`
	Noto         string `json:"noto,omitempty"`
	Twemoji      string `json:"twemoji,omitempty"`
	Wikimedia    string `json:"wikimedia,omitempty"`
}

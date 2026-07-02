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

// Version is the released version of the client library. It is reported as part
// of the User-Agent header sent with every request to the Ipregistry API.
const Version = "1.0.0"

// userAgent is the default value of the User-Agent header sent with requests.
const userAgent = "IpregistryClient/Go/" + Version

# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- `Client.LookupAddr` and `Client.LookupBatchAddr` accept `net/netip.Addr` values as a typed convenience over the
  string-based `Lookup`/`LookupBatch`, validating each address client-side before sending the request.

## [1.0.0] - 2026-07-02
### Added
- Initial release of the official Go client library for the Ipregistry API.
- `Client.Lookup` for single IP address lookups, `Client.LookupOrigin` for origin (requester) lookups returning parsed
  User-Agent data, and `Client.LookupBatch` for resolving many IP addresses at once. Batch lookups transparently split
  slices larger than the API's 1024-address limit into concurrently dispatched sub-requests (configurable via
  `WithMaxBatchSize` and `WithBatchConcurrency`) and reassemble results in input order.
- `Client.ParseUserAgents` for parsing raw User-Agent strings into structured data.
- Batch results (`IPInfoList`, `UserAgentList`) expose per-entry success or failure via `At` and a `range`-compatible
  `All` iterator.
- Context-first API: every method takes a `context.Context` for cancellation and deadlines; the client is safe for
  concurrent use.
- Functional options for configuration: `WithBaseURL`, `WithHTTPClient`, `WithCache`, `WithTimeout`, `WithMaxRetries`,
  `WithRetryInterval`, `WithRetryOnServerError`, `WithRetryOnTooManyRequests`, and `WithUserAgent`.
- Lookup options `WithHostname`, `WithFields`, and `WithParam`.
- Optional in-memory caching (`NewInMemoryCache`) with LRU eviction and TTL, plus a `Cache` interface for custom
  backends. Caching is disabled by default; origin lookups are never cached and batch lookups reuse cached entries.
- Automatic retries with exponential backoff for transient network errors and 5xx responses, honoring the `Retry-After`
  header. Retries on 429 Too Many Requests are disabled by default.
- Typed errors: `*APIError` (with raw `Code` and typed `ErrorCode`) and `*ClientError`, both matchable with `errors.As`.
- `IsBot` helper to skip lookups for crawler traffic.
- Offline unit/behavior tests using `net/http/httptest`, plus opt-in live system tests behind the `integration` build
  tag (run with `IPREGISTRY_API_KEY` set).
- Zero external dependencies (standard library only).

[Unreleased]: https://github.com/ipregistry/ipregistry-go/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/ipregistry/ipregistry-go/releases/tag/v1.0.0

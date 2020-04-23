# openrtb [![GoDoc](https://godoc.org/github.com/mxmCherry/openrtb?status.svg)](https://godoc.org/github.com/mxmCherry/openrtb) [![Build Status](https://travis-ci.org/mxmCherry/openrtb.svg?branch=master)](https://travis-ci.org/mxmCherry/openrtb)

[OpenRTB](https://www.iab.com/guidelines/real-time-bidding-rtb-project/) [v2.5](https://www.iab.com/wp-content/uploads/2016/03/OpenRTB-API-Specification-Version-2-5-FINAL.pdf) types for Go programming language (Golang)

Also includes [OpenRTB](https://www.iab.com/guidelines/real-time-bidding-rtb-project/) [Dynamic Native Ads API Specification Version 1.2](https://www.iab.com/wp-content/uploads/2018/03/OpenRTB-Native-Ads-Specification-Final-1.2.pdf) types:
- [4 Native Ad Request Markup Details](native/request/)
- [5 Native Ad Response Markup Details](native/response/)
- [7 Reference Lists/Enumerations](native/)

**Requires Go 1.8+**

Go 1.8+ is needed for proper `Ext json.RawMessage` marshaling: non-pointer `json.RawMessage` is marshaled as base64 string prior to Go 1.8.

This library uses `json.RawMessage` since [v10.0.0](https://github.com/mxmCherry/openrtb/releases/tag/v10.0.0).

# Using

```bash
go get -u "github.com/mxmCherry/openrtb"
```

```go
import "github.com/mxmCherry/openrtb"
```

This repo follows [semver](http://semver.org/) - see [releases](https://github.com/mxmCherry/openrtb/releases).
Master always contains latest code, so better use some package manager to vendor specific version.

# Guidelines

## Naming convention
- [UpperCamelCase](http://en.wikipedia.org/wiki/CamelCase)
- Capitalized abbreviations (e.g., `AT`, `COPPA`, `PMP` etc.)
- Capitalized `ID` keys

## Types
- Key types should be chosen according to OpenRTB specification (attribute types)
- Numeric types:
	- `int8` - short enums (with values <= 127), boolean-like attributes (like `BidRequest.test`)
	- `int64` - time, duration, length, unbound enums (like `BidRequest.at` - exchange-specific auctions types are > 500)
	- `uint64` - width, height, bitrate etc. (unbound positive numbers)
	- `float64` - coordinates, prices etc.
- Enums:
	- all enums, described in section 5, must be typed with section name singularized (e.g., "5.2 Banner Ad Types" -> `type BannerAdType int8`)
	- all typed enums must have constants for each element, prefixed with type name (e.g., "5.2 Banner Ad Types - XHTML Text Ad (usually mobile)" -> `const BannerAdTypeXHTMLTextAd BannerAdType = 1`)
	- never use `iota` for enum constants
	- section "5.1 Content Categories" should remain untyped and have no constants

## Pointers/omitempty
Pointer | Omitempty | When to use                                                          | Example
------- | --------- | -------------------------------------------------------------------- | ---------------------------------
 no     | no        | _required_ in spec                                                   | `Audio.mimes`
 yes    | yes       | _required_ in spec, but is a part of mutually-exclusive group        | `Imp.{banner,video,audio,native}`
 no     | yes       | zero value (`""`, `0`) is useless / has no meaning                   | `Device.ua`
 yes    | yes       | zero value (`""`, `0`) or value absence (`null`) has special meaning | `Device.{dnt,lmt}`

Using both pointer and `omitempty` is mostly just to save traffic / generate more "canonical" (strict) JSON.

## Documentation ([godoc](https://godoc.org/github.com/mxmCherry/openrtb))
- [Godoc: documenting Go code](http://blog.golang.org/godoc-documenting-go-code)
- Each entity (type, struct key or constant) should be documented
- Comments for entities should be copy-pasted "as-is" from OpenRTB specification (except section 5 - replace "table" with "list" there; ideally, each sentence must be on a new line)

## Code organization
- Each RTB type should be kept in its own file, named after type
- File names are in underscore_case, e.g., `type BidRequest` should be declared in `bid_request.go`
- [go fmt your code](https://blog.golang.org/go-fmt-your-code)
- [EditorConfig](https://editorconfig.org/) (not required, but useful)

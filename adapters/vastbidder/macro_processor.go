package vastbidder

import (
	"bytes"
	"net/url"
	"strings"

	"github.com/golang/glog"
)

const (
	macroPrefix          string = `{` //macro prefix can not be empty
	macroSuffix          string = `}` //macro suffix can not be empty
	macroEscapeSuffix    string = `_ESC`
	macroPrefixLen       int    = len(macroPrefix)
	macroSuffixLen       int    = len(macroSuffix)
	macroEscapeSuffixLen int    = len(macroEscapeSuffix)
)

//Flags to customize macro processing wrappers

//MacroProcessor struct to hold openrtb request and cache values
type MacroProcessor struct {
	bidderMacro IBidderMacro
	mapper      Mapper
	macroCache  map[string]string
	bidderKeys  map[string]string
}

//NewMacroProcessor will process macro's of openrtb bid request
func NewMacroProcessor(bidderMacro IBidderMacro, mapper Mapper) *MacroProcessor {
	return &MacroProcessor{
		bidderMacro: bidderMacro,
		mapper:      mapper,
		macroCache:  make(map[string]string),
	}
}

//SetMacro Adding Custom Macro Manually
func (mp *MacroProcessor) SetMacro(key, value string) {
	mp.macroCache[key] = value
}

//SetBidderKeys will flush and set bidder specific keys
func (mp *MacroProcessor) SetBidderKeys(keys map[string]string) {
	mp.bidderKeys = keys
}

//processKey : returns value of key macro and status found or not
func (mp *MacroProcessor) processKey(key string) (string, bool) {
	var valueCallback *macroCallBack
	var value string
	nEscaping := 0
	tmpKey := key
	found := false

	for {
		//Search in macro cache
		if value, found = mp.macroCache[tmpKey]; found {
			break
		}

		//Search for bidder keys
		if nil != mp.bidderKeys {
			if value, found = mp.bidderKeys[tmpKey]; found {
				//default escaping of bidder keys
				if len(value) > 0 && nEscaping == 0 {
					//escape parameter only if _ESC is not present
					value = url.QueryEscape(value)
				}
				break
			}
		}

		valueCallback, found = mp.mapper[tmpKey]
		if found {
			//found callback function
			value = valueCallback.callback(mp.bidderMacro, tmpKey)

			//checking if default escaping needed or not
			if len(value) > 0 && valueCallback.escape && nEscaping == 0 {
				//escape parameter only if defaultescaping is true and _ESC is not present
				value = url.QueryEscape(value)
			}

			break
		} else if strings.HasSuffix(tmpKey, macroEscapeSuffix) {
			//escaping macro found
			tmpKey = tmpKey[0 : len(tmpKey)-macroEscapeSuffixLen]
			nEscaping++
			continue
		}
		break
	}

	if found {
		if len(value) > 0 {
			if nEscaping > 0 {
				//escaping string nEscaping times
				value = escape(value, nEscaping)
			}
			if nil != valueCallback && valueCallback.cached {
				//cached value if its cached flag is true
				mp.macroCache[key] = value
			}
		}
	}

	return value, found
}

//Process : Substitute macros in input string
func (mp *MacroProcessor) Process(in string) (response string) {
	var out bytes.Buffer
	pos, start, end, size := 0, 0, 0, len(in)

	for pos < size {
		//find macro prefix index
		if start = strings.Index(in[pos:], macroPrefix); -1 == start {
			//[prefix_not_found] append remaining string to response
			out.WriteString(in[pos:])

			//macro prefix not found
			break
		}

		//prefix index w.r.t original string
		start = start + pos

		//append non macro prefix content
		out.WriteString(in[pos:start])

		if (end - macroSuffixLen) <= (start + macroPrefixLen) {
			//string contains {{TEXT_{{MACRO}} -> it should replace it with{{TEXT_MACROVALUE
			//find macro suffix index
			if end = strings.Index(in[start+macroPrefixLen:], macroSuffix); -1 == end {
				//[suffix_not_found] append remaining string to response
				out.WriteString(in[start:])

				// We Found First %% and Not Found Second %% But We are in between of string
				break
			}

			end = start + macroPrefixLen + end + macroSuffixLen
		}

		//get actual macro key by removing macroPrefix and macroSuffix from key itself
		key := in[start+macroPrefixLen : end-macroSuffixLen]

		//process macro
		value, found := mp.processKey(key)
		if found {
			out.WriteString(value)
			pos = end
		} else {
			out.WriteByte(macroPrefix[0])
			pos = start + 1
		}
		//glog.Infof("\nSearch[%d] <start,end,key>: [%d,%d,%s]", count, start, end, key)
	}
	response = out.String()
	glog.V(3).Infof("[MACRO]:in:[%s] replaced:[%s]", in, response)
	return
}

//GetMacroKey will return macro formatted key
func GetMacroKey(key string) string {
	return macroPrefix + key + macroSuffix
}

func escape(str string, n int) string {
	for ; n > 0; n-- {
		str = url.QueryEscape(str)
	}
	return str[:]
}

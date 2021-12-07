package consumable

import (
	"strconv"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
)

/* Turn array of openrtb formats into consumable's code*/
func getSizeCodes(Formats []openrtb2.Format) []int {

	codes := make([]int, 0)
	for _, format := range Formats {
		str := strconv.FormatInt(format.W, 10) + "x" + strconv.FormatInt(format.H, 10)
		if code, ok := sizeMap[str]; ok {
			codes = append(codes, code)
		}
	}
	return codes
}

var sizeMap = map[string]int{
	"120x90": 1,
	// 120x90 is in twice in prebid.js implementation - probably as spacer
	"468x60":  3,
	"728x90":  4,
	"300x250": 5,
	"160x600": 6,
	"120x600": 7,
	"300x100": 8,
	"180x150": 9,
	"336x280": 10,
	"240x400": 11,
	"234x60":  12,
	"88x31":   13,
	"120x60":  14,
	"120x240": 15,
	"125x125": 16,
	"220x250": 17,
	"250x250": 18,
	"250x90":  19,
	"0x0":     20, // TODO: can this be removed - I suspect it's padding in prebid.js impl
	"200x90":  21,
	"300x50":  22,
	"320x50":  23,
	"320x480": 24,
	"185x185": 25,
	"620x45":  26,
	"300x125": 27,
	"800x250": 28,
	// below order is preserved from prebid.js implementation for easy comparison
	"970x90":   77,
	"970x250":  123,
	"300x600":  43,
	"970x66":   286,
	"970x280":  3230,
	"486x60":   429,
	"700x500":  374,
	"300x1050": 934,
	"320x100":  1578,
	"320x250":  331,
	"320x267":  3301,
	"728x250":  2730,
}

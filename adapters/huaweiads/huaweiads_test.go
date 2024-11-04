package huaweiads

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderHuaweiAds, config.Adapter{
		Endpoint: "https://huaweiads.com/adxtest/",
		ExtraAdapterInfo: "{\"pkgNameConvert\":[{\"convertedPkgName\":\"com.example.pkgname1\"," +
			"\"unconvertedPkgNames\":[\"com.example.p1\",\"com.example.p2\"]," +
			"\"unconvertedPkgNameKeyWords\":[\"p3\",\"p4\"]," +
			"\"unconvertedPkgNamePrefixs\":[\"com.example1\",\"com.example2\"]," +
			"\"exceptionPkgNames\":[\"com.example.p7\",\"com.example.p8\"]}," +
			"{\"convertedPkgName\":\"com.example.pkgname2\"," +
			"\"unconvertedPkgNames\":[\"com.example.p9\",\"com.example.p10\"]," +
			"\"unconvertedPkgNameKeyWords\":[\"p11\",\"p12\"]," +
			"\"unconvertedPkgNamePrefixs\":[\"com.example3\",\"com.example4\"]," +
			"\"exceptionPkgNames\":[\"com.example.p15\",\"com.example3.unchanged\"]}]}",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}
	adapterstest.RunJSONBidderTest(t, "huaweiadstest", bidder)
}

func TestExtraInfoDefaultWhenEmpty(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderHuaweiAds, config.Adapter{
		Endpoint:         `https://huaweiads.com/adxtest/`,
		ExtraAdapterInfo: ``,
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderHuaweiAds, _ := bidder.(*adapter)

	assert.Equal(t, bidderHuaweiAds.extraInfo.PkgNameConvert, []pkgNameConvert(nil))
}

func TestExtraInfo1(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderHuaweiAds, config.Adapter{
		Endpoint: `https://huaweiads.com/adxtest/`,
		ExtraAdapterInfo: "{\"pkgNameConvert\":[{\"convertedPkgName\":\"com.example.pkgname\"," +
			"\"unconvertedPkgNames\":[\"com.example.p1\",\"com.example.p2\"]," +
			"\"unconvertedPkgNameKeyWords\":[\"com.example.p3\",\"com.example.p4\"]," +
			"\"unconvertedPkgNamePrefixs\":[\"com.example.p5\",\"com.example.p6\"]," +
			"\"exceptionPkgNames\":[\"com.example.p7\",\"com.example.p8\"]}]}",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderHuaweiAds, _ := bidder.(*adapter)

	assert.Equal(t, bidderHuaweiAds.extraInfo.PkgNameConvert[0].ConvertedPkgName, "com.example.pkgname")
	assert.Equal(t, bidderHuaweiAds.extraInfo.PkgNameConvert[0].UnconvertedPkgNameKeyWords, []string{"com.example.p3", "com.example.p4"})
	assert.Equal(t, bidderHuaweiAds.extraInfo.PkgNameConvert[0].UnconvertedPkgNamePrefixs, []string{"com.example.p5", "com.example.p6"})
	assert.Equal(t, bidderHuaweiAds.extraInfo.PkgNameConvert[0].UnconvertedPkgNames, []string{"com.example.p1", "com.example.p2"})
	assert.Equal(t, bidderHuaweiAds.extraInfo.PkgNameConvert[0].ExceptionPkgNames, []string{"com.example.p7", "com.example.p8"})
}

func TestExtraInfo2(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderHuaweiAds, config.Adapter{
		Endpoint: `https://huaweiads.com/adxtest/`,
		ExtraAdapterInfo: "{\"pkgNameConvert\":[{\"convertedPkgName\":\"com.example.pkgname1\"," +
			"\"unconvertedPkgNames\":[\"com.example.p1\",\"com.example.p2\"]," +
			"\"unconvertedPkgNameKeyWords\":[\"com.example.p3\",\"com.example.p4\"]," +
			"\"unconvertedPkgNamePrefixs\":[\"com.example.p5\",\"com.example.p6\"]," +
			"\"exceptionPkgNames\":[\"com.example.p7\",\"com.example.p8\"]}," +
			"{\"convertedPkgName\":\"com.example.pkgname2\"," +
			"\"unconvertedPkgNames\":[\"com.example.p9\",\"com.example.p10\"]," +
			"\"unconvertedPkgNameKeyWords\":[\"com.example.p11\",\"com.example.p12\"]," +
			"\"unconvertedPkgNamePrefixs\":[\"com.example.p13\",\"com.example.p14\"]," +
			"\"exceptionPkgNames\":[\"com.example.p15\",\"com.example.p16\"]}]}",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderHuaweiAds, _ := bidder.(*adapter)

	assert.Equal(t, bidderHuaweiAds.extraInfo.PkgNameConvert[1].ConvertedPkgName, "com.example.pkgname2")
	assert.Equal(t, bidderHuaweiAds.extraInfo.PkgNameConvert[0].UnconvertedPkgNameKeyWords, []string{"com.example.p3", "com.example.p4"})
	assert.Equal(t, bidderHuaweiAds.extraInfo.PkgNameConvert[1].UnconvertedPkgNamePrefixs, []string{"com.example.p13", "com.example.p14"})
	assert.Equal(t, bidderHuaweiAds.extraInfo.PkgNameConvert[0].UnconvertedPkgNames, []string{"com.example.p1", "com.example.p2"})
	assert.Equal(t, bidderHuaweiAds.extraInfo.PkgNameConvert[1].ExceptionPkgNames, []string{"com.example.p15", "com.example.p16"})
}

func TestExtraInfo3(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderHuaweiAds, config.Adapter{
		Endpoint: `https://huaweiads.com/adxtest/`,
		ExtraAdapterInfo: "{\"pkgNameConvert\":[{\"convertedPkgName\":\"\"," +
			"\"unconvertedPkgNames\":[\"com.example.p1\",\"com.example.p2\"]," +
			"\"unconvertedPkgNameKeyWords\":[\"com.example.p3\",\"com.example.p4\"]," +
			"\"unconvertedPkgNamePrefixs\":[\"com.example.p5\",\"com.example.p6\"]," +
			"\"exceptionPkgNames\":[\"com.example.p7\",\"com.example.p8\"]}]," +
			"\"closeSiteSelectionByCountry\":\"1\"}",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	assert.Equal(t, buildErr.Error(), "invalid extra info: ConvertedPkgName is empty, pls check")
}

func TestExtraInfo4(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderHuaweiAds, config.Adapter{
		Endpoint: `https://huaweiads.com/adxtest/`,
		ExtraAdapterInfo: "{\"pkgNameConvert\":[{\"convertedPkgName\":\"com.example.pkgname1\"," +
			"\"unconvertedPkgNames\":[\"com.example.p1\",\"com.example.p2\"]," +
			"\"unconvertedPkgNameKeyWords\":[\"\",\"com.example.p4\"]," +
			"\"unconvertedPkgNamePrefixs\":[\"com.example.p5\",\"\"]," +
			"\"exceptionPkgNames\":[\"com.example.p7\",\"com.example.p8\"]}]," +
			"\"closeSiteSelectionByCountry\":\"1\"}",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	assert.Equal(t, buildErr.Error(), "invalid extra info: UnconvertedPkgNameKeyWords has a empty keyword, pls check")
}

func TestExtraInfo5(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderHuaweiAds, config.Adapter{
		Endpoint: `https://huaweiads.com/adxtest/`,
		ExtraAdapterInfo: "{\"pkgNameConvert\":[{\"convertedPkgName\":\"com.example.pkgname1\"," +
			"\"unconvertedPkgNames\":[\"com.example.p1\",\"com.example.p2\"]," +
			"\"unconvertedPkgNameKeyWords\":[\"com.example.p3\",\"com.example.p4\"]," +
			"\"unconvertedPkgNamePrefixs\":[\"com.example.p5\",\"\"]," +
			"\"exceptionPkgNames\":[\"com.example.p7\",\"com.example.p8\"]}]," +
			"\"closeSiteSelectionByCountry\":\"1\"}",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	assert.Equal(t, buildErr.Error(), "invalid extra info: UnconvertedPkgNamePrefixs has a empty value, pls check")
}

func TestExtraInfo6(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderHuaweiAds, config.Adapter{
		Endpoint: `https://huaweiads.com/adxtest/`,
		ExtraAdapterInfo: "{\"pkgNameConvert\":[{\"convertedPkgName\":\"com.example.pkgname1\"," +
			"\"unconvertedPkgNames\":[\"com.example.p1\",\"com.example.p2\"]," +
			"\"unconvertedPkgNameKeyWords\":[\"com.example.p3\",\"com.example.p4\"]," +
			"\"unconvertedPkgNamePrefixs\":[\"com.example.p5\",\"com.example.p6\"]," +
			"\"exceptionPkgNames\":[\"com.example.p7\",\"com.example.p8\"]}," +
			"{\"convertedPkgName\":\"com.example.pkgname2\"," +
			"\"unconvertedPkgNames\":[\"com.example.p9\",\"com.example.p10\"]," +
			"\"unconvertedPkgNameKeyWords\":[\"com.example.p11\",\"com.example.p12\"]," +
			"\"unconvertedPkgNamePrefixs\":[\"com.example.p13\",\"com.example.p14\"]," +
			"\"exceptionPkgNames\":[\"com.example.p15\",\"com.example.p16\"]}],\"closeSiteSelectionByCountry\":\"1\"}",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderHuaweiAds, _ := bidder.(*adapter)

	assert.Equal(t, bidderHuaweiAds.extraInfo.CloseSiteSelectionByCountry, "1")
}

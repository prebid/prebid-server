package consumable

import (
	"strconv"
	"time"
)

func retrieveAd(decision decision, unitId int, unitName string, now time.Time) string {

	oad := ""
	if decision.Contents != nil && len(decision.Contents) > 0 {
		oad = decision.Contents[0].Body + createTrackPixelHtml(decision.ImpressionUrl)
	}

	cb := strconv.FormatInt(now.Unix(), 10)
	sUnitId := strconv.Itoa(unitId)
	ad := "<script type=\"text/javascript\">document.write(\"<div id=\"" +
		unitName + "-" + sUnitId + "\">\");</script>" + oad +
		"<script type=\"text/javascript\">document.write(\"</div>\");</script>" +
		"<script type=\"text/javascript\">document.write(\"<div class=\"" + unitName + "\"></div>\");</script>" +
		"<script type=\"text/javascript\">document.write(\"<scr\"+\"ipt type=\"text/javascript\" src=\"https://yummy.consumable.com/" +
		sUnitId + "/" + unitName + "/widget/unit.js?cb=" + cb + "\" charset=\"utf-8\" async></scr\"+\"ipt>\");</script>"

	return ad
}

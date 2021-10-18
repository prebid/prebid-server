package consumable

import (
	netUrl "net/url"
)

/**
 * Creates a snippet of HTML that retrieves the specified `url`
 * Returns    HTML snippet that contains the img src = set to `url`
 */
func createTrackPixelHtml(url *string) string {
	if url == nil {
		return ""
	}

	escapedUrl := netUrl.QueryEscape(*url)
	img := "<div style=\"position:absolute;left:0px;top:0px;visibility:hidden;\">" +
		"<img src=\"" + escapedUrl + "\"></div>"
	return img
}

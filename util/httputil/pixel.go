package httputil

import _ "embed"

type Pixel struct {
	Content     []byte
	ContentType string
}

//go:embed pixel.png
var pixelContent []byte
var Pixel1x1PNG = Pixel{
	Content:     pixelContent,
	ContentType: "image/png",
}

package framework_for_imgui

import (
	_ "embed"

	"github.com/inkyblackness/imgui-go/v4"
)

var (
	//go:embed fonts/PixelMplus12-Regular.ttf
	pixelMPlus12 []byte
)

func SetupFont(io imgui.IO) []imgui.Font {
	fonts := io.Fonts()

	// fonts.AddFontDefault()
	var fontsData []imgui.Font
	fontsData = append(fontsData, fonts.AddFontFromMemoryTTFV(pixelMPlus12, 32.0, imgui.DefaultFontConfig, fonts.GlyphRangesJapanese()))
	fontsData = append(fontsData, fonts.AddFontFromMemoryTTFV(pixelMPlus12, 10.0, imgui.DefaultFontConfig, fonts.GlyphRangesJapanese()))

	return fontsData
}

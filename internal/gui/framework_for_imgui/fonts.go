package framework_for_imgui

import (
	_ "embed"

	"github.com/inkyblackness/imgui-go/v4"
)

var (
	//go:embed fonts/PixelMplus10-Regular.ttf
	pixelMPlus10 []byte
)

func SetupFont(io imgui.IO) []imgui.Font {
	fonts := io.Fonts()

	var fontsData []imgui.Font
	config := imgui.NewFontConfig()
	defer config.Delete()

	// fontsData = append(fontsData, fonts.AddFontFromMemoryTTFV(pixelMPlus10, 32.0, config, fonts.GlyphRangesJapanese()))
	// fontsData = append(fontsData, fonts.AddFontFromMemoryTTFV(pixelMPlus10, 28.0, config, fonts.GlyphRangesJapanese()))
	// fontsData = append(fontsData, fonts.AddFontFromMemoryTTFV(pixelMPlus10, 22.0, config, fonts.GlyphRangesJapanese()))
	// fontsData = append(fontsData, fonts.AddFontFromMemoryTTFV(pixelMPlus10, 18.0, config, fonts.GlyphRangesJapanese()))
	// fontsData = append(fontsData, fonts.AddFontFromMemoryTTFV(pixelMPlus10, 14.0, config, fonts.GlyphRangesJapanese()))

	config.SetOversampleH(1)
	glyphRange := fonts.GlyphRangesDefault()
	fontsData = append(fontsData, fonts.AddFontFromMemoryTTFV(pixelMPlus10, 28.0, config, glyphRange))
	fontsData = append(fontsData, fonts.AddFontFromMemoryTTFV(pixelMPlus10, 22.0, config, glyphRange))
	fontsData = append(fontsData, fonts.AddFontFromMemoryTTFV(pixelMPlus10, 20.0, config, glyphRange))

	return fontsData
}

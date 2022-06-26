package main

import (
	"fmt"

	"github.com/inkyblackness/imgui-go/v4"
)

const (
	// A0
	DEFUALT_PIANO_LOWEST_PITCH = 21
	// C8
	DEFUALT_PIANO_HIGHEST_PITCH = 108
)

const (
	_ = iota
	KEY_WHITE
	KEY_BLACK
)

type KeyInfo struct {
	pitch   string
	isWhite bool
	index   int
	posX1   float32
	posY1   float32
	posX2   float32
	posY2   float32
}

var keyArray []*KeyInfo
var Pitch2KeyIndexTable map[string]int

var blackKeyColor imgui.PackedColor = imgui.PackedColorFromVec4(imgui.Vec4{0, 0, 0, 1.0})
var whiteKeyColor imgui.PackedColor = imgui.PackedColorFromVec4(imgui.Vec4{1.0, 1.0, 1.0, 1.0})
var fillKeyColor imgui.PackedColor = imgui.PackedColorFromVec4(imgui.Vec4{0.392, 0.584, 0.929, 1.0})
var keyGapColor imgui.PackedColor = imgui.PackedColorFromVec4(imgui.Vec4{0.2, 0.2, 0.2, 1.0})

func initPianoKeyboard() {
	createKeyMap()

	Pitch2KeyIndexTable = make(map[string]int)
	for i, v := range keyArray {
		Pitch2KeyIndexTable[v.pitch] = i
	}
}

func DrawPianoKeyboard(list *imgui.DrawList, pos imgui.Vec2, fillIndex int) {
	// count total white key
	var totalWhiteKey int = 0
	for _, v := range keyArray {
		if v.isWhite == true {
			totalWhiteKey++
		}
	}

	keyWidth := imgui.WindowContentRegionWidth() / float32(totalWhiteKey)
	whiteKeyHeight := keyWidth * 4.0
	blackKeyHeight := whiteKeyHeight * 0.6666

	var currentX float32 = 0
	// draw white keys
	for i := 0; i < len(keyArray); i++ {
		if keyArray[i].isWhite {
			keyArray[i].posX1 = pos.X + currentX
			keyArray[i].posY1 = pos.Y
			keyArray[i].posX2 = pos.X + currentX + keyWidth
			keyArray[i].posY2 = pos.Y + whiteKeyHeight

			currentX += keyWidth
		} else {
			keyArray[i].posX1 = pos.X + currentX - (keyWidth / 4)
			keyArray[i].posY1 = pos.Y
			keyArray[i].posX2 = pos.X + currentX - (keyWidth / 4) + (keyWidth / 2)
			keyArray[i].posY2 = pos.Y + blackKeyHeight
		}
	}

	var col imgui.PackedColor
	for i := 0; i < len(keyArray); i++ {
		// draw white keys
		if keyArray[i].isWhite {
			key := keyArray[i]

			col = whiteKeyColor
			if i == fillIndex {
				col = fillKeyColor
			}

			list.AddRectFilledV(
				imgui.Vec2{
					X: key.posX1,
					Y: key.posY1,
				},
				imgui.Vec2{
					X: key.posX2,
					Y: key.posY2,
				},
				col, 0, imgui.DrawFlagsRoundCornersNone,
			)

			list.AddRectV(
				imgui.Vec2{
					X: key.posX1,
					Y: key.posY1,
				},
				imgui.Vec2{
					X: key.posX2,
					Y: key.posY2,
				},
				keyGapColor, 0, imgui.DrawCornerFlagsNone, 1,
			)
		}
	}

	for i := 0; i < len(keyArray); i++ {
		if keyArray[i].isWhite == false {
			key := keyArray[i]

			col = blackKeyColor
			if i == fillIndex {
				col = fillKeyColor
			}

			list.AddRectFilledV(
				imgui.Vec2{
					X: key.posX1,
					Y: key.posY1,
				},
				imgui.Vec2{
					X: key.posX2,
					Y: key.posY2,
				},
				col, 0, imgui.DrawFlagsRoundCornersNone,
			)

			list.AddRectV(
				imgui.Vec2{
					X: key.posX1,
					Y: key.posY1,
				},
				imgui.Vec2{
					X: key.posX2,
					Y: key.posY2,
				},
				keyGapColor, 0, imgui.DrawCornerFlagsNone, 1,
			)
		}
	}
}

func createKeyMap() {
	index := 0
	for i := DEFUALT_PIANO_LOWEST_PITCH; i < DEFUALT_PIANO_HIGHEST_PITCH; i++ {
		keyInfo := &KeyInfo{
			pitch:   getKeyPitch(i),
			index:   i - DEFUALT_PIANO_LOWEST_PITCH,
			isWhite: getKeyType(i) == KEY_WHITE,
		}
		keyArray = append(keyArray, keyInfo)
		index++
	}
}

func getKeyType(pitch int) int {
	switch pitch % 12 {
	case 0, 5:
		// C, F
		return KEY_WHITE
	case 4, 11:
		// E, B
		return KEY_WHITE
	case 2, 7, 9:
		// D, G, A
		return KEY_WHITE
	case 1, 3, 6, 8, 10:
		// C#, D#, F#, G#, A#
		return KEY_BLACK
	}

	// NOT REACH
	return KEY_WHITE
}

func getKeyPitch(pitch int) string {
	var codeArray [12]string = [12]string{
		"C",
		"C#",
		"D",
		"D#",
		"E",
		"F",
		"F#",
		"G",
		"G#",
		"A",
		"A#",
		"B",
	}
	codeStr := codeArray[pitch%12]

	return fmt.Sprintf("%s%d", codeStr, (pitch/12)-1)
}

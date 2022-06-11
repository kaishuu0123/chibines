// refs: github.com/libretro/Mesen
package toynes

import (
	"image"
	"log"
)

const (
	SCREEN_WIDTH          = 256
	SCREEN_HEIGHT         = 240
	PIXEL_COUNT           = 256 * 240
	OAM_DECAY_CYCLE_COUNT = 3000
)

type TileInfo struct {
	lowByte       byte
	highByte      byte
	paletteOffset uint32
	tileAddr      uint16

	absoluteTileAddr int32
	offsetY          uint8
}

type SpriteInfo struct {
	*TileInfo

	horizontalMirror   bool
	backgroundPriority bool
	spriteX            byte

	verticalMirror bool
}

type PPUControlFlags struct {
	verticalWrite         bool
	spritePatternAddr     uint16
	backgroundPatternAddr uint16
	largeSprites          bool
	vblank                bool

	grayscale         bool
	backgroundMask    bool
	spriteMask        bool
	backgroundEnabled bool
	spritesEnabled    bool
	intensifyRed      bool
	intensifyGreen    bool
	intensifyBlue     bool
}

type PPUStatusFlags struct {
	SpriteOverflow bool
	Sprite0Hit     bool
	VerticalBlank  bool
}

type PPUState struct {
	Control         byte
	Mask            byte
	Status          byte
	SpriteRAMAddr   uint32
	VideoRAMAddr    uint16
	XScroll         uint8
	TmpVideoRAMAddr uint16
	WriteToggle     bool

	HighBitShift uint16
	LowBitShift  uint16
}

type PPU struct {
	console *Console // reference to parent object

	state PPUState

	ScanLine         int    // 0-261, 0-239=visible, 240=post, 241-260=vblank, 261=pre
	Cycle            uint32 // 0-340
	Frame            uint64 // frame counter
	masterClock      uint64
	memoryReadBuffer byte

	// storage variables
	paletteRAM         [32]byte  // 0x20
	spriteRAM          [256]byte // 0x100
	secondarySpriteRAM [32]byte  // 0x20
	hasSprite          [257]bool
	front              *image.RGBA
	back               *image.RGBA

	standardVblankEnd   uint16
	standardNMIScanline uint16
	vblankEnd           uint16
	nmiScanLine         uint16

	flags       PPUControlFlags
	statusFlags PPUStatusFlags

	intensifyColorBits uint16
	paletteRAMMask     byte
	lastUpdatedPixel   int32
	lastSprite         *SpriteInfo

	ppuBusAddress uint16
	currentTile   TileInfo
	nextTile      TileInfo
	previousTile  TileInfo

	spriteTiles      [64]SpriteInfo
	spriteCount      uint32
	secondaryOAMAddr uint32
	sprite0Visible   bool

	firstVisibleSpriteAddr byte
	lastVisibleSpriteAddr  byte
	spriteIndex            uint32

	openBus           byte
	openBusDecayStamp [8]int32
	ignoreVRAMRead    uint32

	oamCopyBuffer      byte
	spriteInRange      bool
	sprite0Added       bool
	spriteAddrH        byte
	spriteAddrL        byte
	oamCopyDone        bool
	overflowBugCounter byte

	needStateUpdate      bool
	renderingEnabled     bool
	prevRenderingEnabled bool
	preventVBLFlag       bool

	updateVRAMAddr      uint16
	updateVRAMAddrDelay byte

	minimumDrawBGCycle             uint32
	minimumDrawSpriteCycle         uint32
	minimumDrawSpriteStandardCycle uint32

	oamDecayCycles [64]uint64 // 0x40
	enableOAMDecay bool
	corruptOAMRow  [32]bool
}

func NewPPU(console *Console) *PPU {
	ppu := PPU{console: console}
	ppu.front = image.NewRGBA(image.Rect(0, 0, 256, 240))
	ppu.back = image.NewRGBA(image.Rect(0, 0, 256, 240))

	var powerupPalette [32]byte = [32]byte{
		0x09, 0x01, 0x00, 0x01, 0x00, 0x02, 0x02, 0x0D,
		0x08, 0x10, 0x08, 0x24, 0x00, 0x00, 0x04, 0x2C,
		0x09, 0x01, 0x34, 0x03, 0x00, 0x04, 0x00, 0x14,
		0x08, 0x3A, 0x00, 0x02, 0x00, 0x20, 0x2C, 0x08,
	}
	copy(ppu.paletteRAM[:], powerupPalette[:])

	for i := 0; i < len(ppu.spriteTiles); i++ {
		ppu.spriteTiles[i].TileInfo = &TileInfo{}
	}

	for i := 0; i < len(ppu.spriteRAM); i++ {
		ppu.spriteRAM[i] = 0x00
	}

	for i := 0; i < len(ppu.secondarySpriteRAM); i++ {
		ppu.secondarySpriteRAM[i] = 0x00
	}

	return &ppu
}

func (ppu *PPU) Reset() {
	ppu.paletteRAMMask = 0x3F
	ppu.lastUpdatedPixel = -1

	// First execution will be cycle 0, scanline 0
	ppu.ScanLine = -1
	ppu.Cycle = 340

	ppu.Frame = 1

	ppu.nmiScanLine = 241
	ppu.vblankEnd = 260
	ppu.standardNMIScanline = 241
	ppu.standardVblankEnd = 260

	ppu.UpdateMinimumDrawCycles()
}

func (ppu *PPU) UpdateGrayscaleAndIntensifyBits() {
	if ppu.ScanLine < 0 || ppu.ScanLine > int(ppu.nmiScanLine) {
		return
	}

	var pixelNumber int
	if ppu.ScanLine >= 240 {
		pixelNumber = 61439
	} else if ppu.Cycle < 3 {
		pixelNumber = (ppu.ScanLine << 8) - 1
	} else if ppu.Cycle <= 258 {
		pixelNumber = (ppu.ScanLine << 8) + int(ppu.Cycle) - 3
	} else {
		pixelNumber = (ppu.ScanLine << 8) + 255
	}

	if ppu.paletteRAMMask == 0x3F && ppu.intensifyColorBits == 0 {
		ppu.lastUpdatedPixel = int32(pixelNumber)
		return
	}

	if ppu.lastUpdatedPixel < int32(pixelNumber) {
		// uint16_t *out = _currentOutputBuffer + _lastUpdatedPixel + 1;
		// while(_lastUpdatedPixel < pixelNumber) {
		// 	*out = (*out & _paletteRamMask) | _intensifyColorBits;
		// 	out++;
		// 	_lastUpdatedPixel++;
		// }
		for ppu.lastUpdatedPixel < int32(pixelNumber) {
			ppu.lastUpdatedPixel++
		}
	}
}

func (ppu *PPU) UpdateMinimumDrawCycles() {
	if ppu.flags.backgroundEnabled {
		if ppu.flags.backgroundMask {
			ppu.minimumDrawBGCycle = 0
		} else {
			ppu.minimumDrawBGCycle = 8
		}
	} else {
		ppu.minimumDrawBGCycle = 300
	}

	if ppu.flags.spritesEnabled {
		if ppu.flags.spriteMask {
			ppu.minimumDrawSpriteCycle = 0
		} else {
			ppu.minimumDrawSpriteCycle = 8
		}
	} else {
		ppu.minimumDrawSpriteCycle = 300
	}

	if ppu.flags.spritesEnabled {
		if ppu.flags.spriteMask {
			ppu.minimumDrawSpriteStandardCycle = 0
		} else {
			ppu.minimumDrawSpriteStandardCycle = 8
		}
	} else {
		ppu.minimumDrawSpriteStandardCycle = 300
	}
}

//Taken from http://wiki.nesdev.com/w/index.php/The_skinny_on_NES_scrolling#Tile_and_attribute_fetching
func (ppu *PPU) GetNameTableAddr() uint16 {
	return 0x2000 | (ppu.state.VideoRAMAddr & 0x0FFF)
}

//Taken from http://wiki.nesdev.com/w/index.php/The_skinny_on_NES_scrolling#Tile_and_attribute_fetching
func (ppu *PPU) GetAttributeAddr() uint16 {
	return 0x23C0 | (ppu.state.VideoRAMAddr & 0x0C00) | ((ppu.state.VideoRAMAddr >> 4) & 0x38) | ((ppu.state.VideoRAMAddr >> 2) & 0x07)
}

func (ppu *PPU) UpdateStatusFlag() {
	var result byte
	if ppu.statusFlags.SpriteOverflow {
		result |= (1 << 5)
	}
	if ppu.statusFlags.Sprite0Hit {
		result |= (1 << 6)
	}
	if ppu.statusFlags.VerticalBlank {
		result |= (1 << 7)
	}
	ppu.state.Status = result

	ppu.statusFlags.VerticalBlank = false
	ppu.console.CPU.ClearNMIFlag()

	if ppu.ScanLine == int(ppu.nmiScanLine) && ppu.Cycle == 0 {
		ppu.preventVBLFlag = true
	}
}

func (ppu *PPU) UpdateVideoRAMAddr() {
	if ppu.ScanLine >= 240 || ppu.IsRenderingEnabled() == false {
		var v uint16
		if ppu.flags.verticalWrite {
			v = 32
		} else {
			v = 1
		}
		ppu.state.VideoRAMAddr = (ppu.state.VideoRAMAddr + v) & 0x7FFF
		ppu.SetBusAddress(ppu.state.VideoRAMAddr & 0x3FFF)
	} else {
		ppu.IncHorizontalScrolling()
		ppu.IncVerticalScrolling()
	}
}

func (ppu *PPU) ProcessTmpAddrScrollGlitch(normalAddr uint16, value uint16, mask uint16) {
	ppu.state.TmpVideoRAMAddr = normalAddr
	// if(_cycle == 257 && _settings->CheckFlag(EmulationFlags::EnablePpu2000ScrollGlitch) && _scanline < 240 && IsRenderingEnabled()) {
	// 	//Use open bus to set some parts of V (glitch that occurs when writing to $2000/$2005/$2006 on cycle 257)
	// 	_state.VideoRamAddr = (_state.VideoRamAddr & ~mask) | (value & mask);
	// }
}

func (ppu *PPU) ReadPaletteRAM(addr uint16) byte {
	addr &= 0x1F
	if addr == 0x10 || addr == 0x14 || addr == 0x18 || addr == 0x1C {
		addr &= ^uint16(0x10)
	}
	return ppu.paletteRAM[addr]
}

func (ppu *PPU) WritePaletteRAM(addr uint16, value byte) {
	addr &= 0x1F
	value &= 0x3F
	if addr == 0x00 || addr == 0x10 {
		ppu.paletteRAM[0x00] = value
		ppu.paletteRAM[0x10] = value
	} else if addr == 0x04 || addr == 0x14 {
		ppu.paletteRAM[0x04] = value
		ppu.paletteRAM[0x14] = value
	} else if addr == 0x08 || addr == 0x18 {
		ppu.paletteRAM[0x08] = value
		ppu.paletteRAM[0x18] = value
	} else if addr == 0x0C || addr == 0x1C {
		ppu.paletteRAM[0x0C] = value
		ppu.paletteRAM[0x1C] = value
	} else {
		ppu.paletteRAM[addr] = value
	}
}

func (ppu *PPU) ReadRAM(addr uint16) byte {
	openBusMask := byte(0xFF)
	returnValue := byte(0)

	switch addr & 0x07 {
	case 0:
		// NOTHING DONE
	case 1:
		// NOTHING DONE
	case 2:
		// PPU STATUS
		ppu.state.WriteToggle = false
		ppu.UpdateStatusFlag()
		returnValue = ppu.state.Status
		openBusMask = 0x1F

		// ppu.ProcessStatusRegOpenBus(openBusMask, returnValue)
	case 3:
		// NOTHING DONE
	case 4:
		// OAM DATA
		if ppu.ScanLine <= 239 && ppu.IsRenderingEnabled() {
			if ppu.Cycle >= 257 && ppu.Cycle <= 320 {
				var step byte
				if ((ppu.Cycle - 257) % 8) > 3 {
					step = 3
				} else {
					step = byte((ppu.Cycle - 257) % 8)
				}

				ppu.secondaryOAMAddr = (ppu.Cycle-257)/8*4 + uint32(step)
				ppu.oamCopyBuffer = ppu.secondarySpriteRAM[ppu.secondaryOAMAddr]
			}

			returnValue = ppu.oamCopyBuffer
		} else {
			returnValue = ppu.ReadSpriteRAM(byte(ppu.state.SpriteRAMAddr))
		}
		openBusMask = 0x00
	case 5:
		// NOTHING DONE
	case 6:
		// NOTHING DONE
	case 7:
		// PPU DATA
		if ppu.ignoreVRAMRead > 0 {
			openBusMask = 0xFF
		} else {
			returnValue = ppu.memoryReadBuffer
			ppu.memoryReadBuffer = ppu.ReadVRAM(ppu.ppuBusAddress&0x3FFF, MemoryRead)

			// DisablePaletteRead
			if (ppu.ppuBusAddress & 0x3FFF) >= 0x3F00 {
				returnValue = ppu.ReadPaletteRAM(ppu.ppuBusAddress) | (ppu.openBus & 0xC0)
				openBusMask = 0xC0
			} else {
				openBusMask = 0x00
			}

			ppu.UpdateVideoRAMAddr()
			ppu.ignoreVRAMRead = 6
			ppu.needStateUpdate = true
		}
	default:
		log.Fatalf("Unknown PPU ReadRAM Addr: %X\n", addr)
	}
	return ppu.ApplyOpenBus(openBusMask, returnValue)
}

func (ppu *PPU) SetControlRegister(value byte) {
	ppu.state.Control = value

	nameTable := (ppu.state.Control & 0x03)

	normalAddr := (ppu.state.TmpVideoRAMAddr & ^uint16(0x0C00)) | (uint16(nameTable) << 10)
	ppu.ProcessTmpAddrScrollGlitch(normalAddr, (uint16(ppu.console.CPU.bus.openBus) << 10), 0x0400)

	ppu.flags.verticalWrite = (ppu.state.Control & 0x04) == 0x04
	if (ppu.state.Control & 0x08) == 0x08 {
		ppu.flags.spritePatternAddr = 0x1000
	} else {
		ppu.flags.spritePatternAddr = 0x0000
	}
	if (ppu.state.Control & 0x10) == 0x10 {
		ppu.flags.backgroundPatternAddr = 0x1000
	} else {
		ppu.flags.backgroundPatternAddr = 0x0000
	}
	ppu.flags.largeSprites = (ppu.state.Control & 0x20) == 0x20
	ppu.flags.vblank = (ppu.state.Control & 0x80) == 0x80

	if ppu.flags.vblank == false {
		ppu.console.CPU.ClearNMIFlag()
	} else if ppu.flags.vblank && ppu.statusFlags.VerticalBlank {
		ppu.console.CPU.SetNMIFlag()
	}
}

func (ppu *PPU) SetMaskRegister(value byte) {
	ppu.state.Mask = value
	ppu.flags.grayscale = (ppu.state.Mask & 0x01) == 0x01
	ppu.flags.backgroundMask = (ppu.state.Mask & 0x02) == 0x02
	ppu.flags.spriteMask = (ppu.state.Mask & 0x04) == 0x04
	ppu.flags.backgroundEnabled = (ppu.state.Mask & 0x08) == 0x08
	ppu.flags.spritesEnabled = (ppu.state.Mask & 0x10) == 0x10
	ppu.flags.intensifyBlue = (ppu.state.Mask & 0x80) == 0x80

	if ppu.renderingEnabled != (ppu.flags.backgroundEnabled || ppu.flags.spritesEnabled) {
		ppu.needStateUpdate = true
	}

	ppu.UpdateMinimumDrawCycles()
	ppu.UpdateGrayscaleAndIntensifyBits()

	if ppu.flags.grayscale {
		ppu.paletteRAMMask = 0x30
	} else {
		ppu.paletteRAMMask = 0x3F
	}

	ppu.flags.intensifyRed = (ppu.state.Mask & 0x20) == 0x20
	ppu.flags.intensifyGreen = (ppu.state.Mask & 0x40) == 0x40
	ppu.intensifyColorBits = uint16(value&0xE0) << 1
}

func (ppu *PPU) WriteRAM(addr uint16, value byte) {
	if addr != 0x4014 {
		ppu.SetOpenBus(0xFF, value)
	}

	if addr == 0x4014 {
		// OAM DMA
		ppu.console.CPU.RunDMATransfer(value)
		return
	}

	switch addr & 0x07 {
	case 0:
		// PPU CTRL
		ppu.SetControlRegister(value)
	case 1:
		// PPU MASK
		ppu.SetMaskRegister(value)
	case 2:
		// PPU STATUS (read only)
	case 3:
		// OAM ADDR
		ppu.state.SpriteRAMAddr = uint32(value)
	case 4:
		// OAM DATA
		if ppu.ScanLine >= 240 {
			if (ppu.state.SpriteRAMAddr & 0x03) == 0x02 {
				value &= 0xE3
			}
			ppu.WriteSpriteRAM(byte(ppu.state.SpriteRAMAddr), value)
			ppu.state.SpriteRAMAddr = (ppu.state.SpriteRAMAddr + 1) & 0xFF
		} else {
			ppu.state.SpriteRAMAddr = (ppu.state.SpriteRAMAddr + 4) & 0xFF
		}
	case 5:
		// PPU SCROLL
		if ppu.state.WriteToggle {
			ppu.state.TmpVideoRAMAddr = (ppu.state.TmpVideoRAMAddr & ^uint16(0x73E0)) | (uint16(value&0xF8) << 2) | (uint16(value&0x07) << 12)
		} else {
			ppu.state.XScroll = value & 0x07

			newAddr := uint16((ppu.state.TmpVideoRAMAddr) & ^uint16(0x001F)) | (uint16(value) >> 3)
			ppu.ProcessTmpAddrScrollGlitch(newAddr, (uint16(ppu.console.CPU.bus.openBus) << 10), 0x001F)
		}
		ppu.state.WriteToggle = !ppu.state.WriteToggle
	case 6:
		// PPU ADDR
		if ppu.state.WriteToggle {
			ppu.state.TmpVideoRAMAddr = (ppu.state.TmpVideoRAMAddr & ^uint16(0x00FF)) | uint16(value)

			ppu.needStateUpdate = true
			ppu.updateVRAMAddrDelay = 3
			ppu.updateVRAMAddr = ppu.state.TmpVideoRAMAddr
		} else {
			newAddr := (ppu.state.TmpVideoRAMAddr & ^uint16(0xFF00)) | (uint16(value&0x3F) << 8)
			ppu.ProcessTmpAddrScrollGlitch(newAddr, (uint16(ppu.console.CPU.bus.openBus) >> 3), 0x0C00)
		}
		ppu.state.WriteToggle = !ppu.state.WriteToggle
	case 7:
		// PPU DATA
		if (ppu.ppuBusAddress & 0x3FFF) >= 0x3F00 {
			ppu.WritePaletteRAM(ppu.ppuBusAddress, value)
		} else {
			if ppu.ScanLine >= 240 || ppu.IsRenderingEnabled() == false {
				ppu.console.Cartridge.Mapper.WriteVRAM(ppu.ppuBusAddress&0x3FFF, value)
			} else {
				ppu.console.Cartridge.Mapper.WriteVRAM(ppu.ppuBusAddress&0x3FFF, byte(ppu.ppuBusAddress&0xFF))
			}
		}
		ppu.UpdateVideoRAMAddr()
	default:
		log.Fatalf("Unknown PPU ReadRAM Addr: %X\n", addr)
	}
}

func (ppu *PPU) ReadVRAM(addr uint16, opeType MemoryOperationType) byte {
	ppu.SetBusAddress(addr)
	return ppu.console.Cartridge.Mapper.ReadVRAM(addr)
}

func (ppu *PPU) WriteVRAM(addr uint16, value byte) {
	ppu.SetBusAddress(addr)
	ppu.console.Cartridge.Mapper.WriteVRAM(addr, value)
}

func (ppu *PPU) ReadSpriteRAM(addr byte) byte {
	if ppu.enableOAMDecay == false {
		return ppu.spriteRAM[addr]
	} else {
		elapsedCycles := uint64(ppu.console.CPU.cycleCount - ppu.oamDecayCycles[addr>>3])

		if elapsedCycles <= OAM_DECAY_CYCLE_COUNT {
			ppu.oamDecayCycles[addr>>3] = ppu.console.CPU.cycleCount
			return ppu.spriteRAM[addr]
		} else {
			// If this 8-byte row hasn't been read/written to in over 3000 cpu cycles (~1.7ms), return 0x10 to simulate decay
			return 0x10
		}
	}
}

func (ppu *PPU) WriteSpriteRAM(addr byte, value byte) {
	ppu.spriteRAM[addr] = value
	if ppu.enableOAMDecay {
		ppu.oamDecayCycles[addr>>3] = ppu.console.CPU.cycleCount
	}
}

func (ppu *PPU) LoadTileInfo() {
	if ppu.IsRenderingEnabled() {
		switch ppu.Cycle & 0x07 {
		case 1:
			ppu.previousTile = ppu.currentTile
			ppu.currentTile = ppu.nextTile

			ppu.state.LowBitShift |= uint16(ppu.nextTile.lowByte)
			ppu.state.HighBitShift |= uint16(ppu.nextTile.highByte)

			tileIndex := ppu.ReadVRAM(ppu.GetNameTableAddr(), PPURenderingRead)
			ppu.nextTile.tileAddr = (uint16(tileIndex) << 4) | (ppu.state.VideoRAMAddr >> 12) | ppu.flags.backgroundPatternAddr
			// XXX: HD PPU
			// ppu.nextTile.OffsetY
		case 3:
			shift := ((ppu.state.VideoRAMAddr >> 4) & 0x04) | (ppu.state.VideoRAMAddr & 0x02)
			ppu.nextTile.paletteOffset = ((uint32(ppu.ReadVRAM(ppu.GetAttributeAddr(), PPURenderingRead)) >> uint32(shift)) & 0x03) << 2
		case 5:
			ppu.nextTile.lowByte = ppu.ReadVRAM(ppu.nextTile.tileAddr, PPURenderingRead)
			// XXX: HD PPU
			// ppu.nextTile.AbsoluteTileAddr
		case 7:
			ppu.nextTile.highByte = ppu.ReadVRAM(ppu.nextTile.tileAddr+8, PPURenderingRead)
		}
	}
}

func (ppu *PPU) LoadSprite(spriteY, tileIndex, attributes, spriteX byte, extraSprite bool) {
	backgroundPriority := (attributes & 0x20) == 0x20
	horizontalMirror := (attributes & 0x40) == 0x40
	verticalMirror := (attributes & 0x80) == 0x80

	var tileAddr uint16
	var lineOffset byte

	if verticalMirror {
		var v byte
		if ppu.flags.largeSprites {
			v = 15
		} else {
			v = 7
		}
		lineOffset = v - byte(ppu.ScanLine-int(spriteY))
	} else {
		lineOffset = byte(ppu.ScanLine - int(spriteY))
	}

	if ppu.flags.largeSprites {
		var v uint16
		if (tileIndex & 0x01) == 0x01 {
			v = 0x1000
		} else {
			v = 0x0000
		}
		if lineOffset >= 8 {
			tileAddr = v | (((uint16(tileIndex) & ^uint16(0x01)) << 4) + (uint16(lineOffset) + 8))
		} else {
			tileAddr = v | (((uint16(tileIndex) & ^uint16(0x01)) << 4) + uint16(lineOffset))
		}
	} else {
		tileAddr = ((uint16(tileIndex) << 4) | ppu.flags.spritePatternAddr) + uint16(lineOffset)
	}

	fetchLastSprite := true
	if (ppu.spriteIndex < ppu.spriteCount || extraSprite) && spriteY < 240 {
		var info *SpriteInfo = &ppu.spriteTiles[ppu.spriteIndex]
		info.backgroundPriority = backgroundPriority
		info.horizontalMirror = horizontalMirror
		info.verticalMirror = verticalMirror
		info.paletteOffset = uint32((attributes&0x03)<<2) | 0x10

		if extraSprite {
			//Use DebugReadVRAM for extra sprites to prevent side-effects.
			// info.LowByte = _console->GetMapper()->DebugReadVRAM(tileAddr);
			// info.HighByte = _console->GetMapper()->DebugReadVRAM(tileAddr + 8);
		} else {
			fetchLastSprite = false
			info.lowByte = ppu.ReadVRAM(tileAddr, PPURenderingRead)
			info.highByte = ppu.ReadVRAM(tileAddr+8, PPURenderingRead)
		}
		info.tileAddr = tileAddr
		// info.absoluteTileAddr = ppu.console.Cartridge.Mapper.ToAbsoluteChrAddress(tileAddr)
		info.offsetY = lineOffset
		info.spriteX = spriteX

		if ppu.ScanLine >= 0 {
			for i := 0; i < 8 && uint16(spriteX)+uint16(i)+1 < uint16(257); i++ {
				ppu.hasSprite[uint16(spriteX)+uint16(i)+1] = true
			}
		}
	}

	if fetchLastSprite {
		lineOffset = 0
		tileIndex = 0xFF
		if ppu.flags.largeSprites {
			var v uint16
			if (tileIndex & 0x01) == 0x01 {
				v = 0x1000
			} else {
				v = 0x0000
			}
			if lineOffset >= 8 {
				tileAddr = v | ((uint16(tileIndex) & ^uint16(0x01)) << 4) + (uint16(lineOffset) + 8)
			} else {
				tileAddr = v | ((uint16(tileIndex) & ^uint16(0x01)) << 4) + (uint16(lineOffset))
			}
		} else {
			tileAddr = ((uint16(tileIndex) << 4) | ppu.flags.spritePatternAddr) + uint16(lineOffset)
		}

		ppu.ReadVRAM(tileAddr, PPURenderingRead)
		ppu.ReadVRAM(tileAddr+8, PPURenderingRead)
	}

	ppu.spriteIndex++
}

// XXX: Not Implemented yet
func (ppu *PPU) LoadExtraSprites() {
}

func (ppu *PPU) LoadSpriteTileInfo() {
	index := ppu.spriteIndex * 4

	spriteY := ppu.secondarySpriteRAM[index+0]
	spriteIndex := ppu.secondarySpriteRAM[index+1]
	spriteAttr := ppu.secondarySpriteRAM[index+2]
	spriteX := ppu.secondarySpriteRAM[index+3]

	ppu.LoadSprite(spriteY, spriteIndex, spriteAttr, spriteX, false)
}

func (ppu *PPU) IsRenderingEnabled() bool {
	return ppu.renderingEnabled
}

func (ppu *PPU) ApplyOpenBus(mask, value byte) byte {
	ppu.SetOpenBus(^mask, value)
	return value | (ppu.openBus & mask)
}

func (ppu *PPU) SetOpenBus(mask, value byte) {
	if mask == 0xFF {
		ppu.openBus = value
		for i := 0; i < 8; i++ {
			ppu.openBusDecayStamp[i] = int32(ppu.Frame)
		}
	} else {
		bus := uint16(ppu.openBus) << 8
		for i := 0; i < 8; i++ {
			bus >>= 1
			if (mask & 0x01) == 0x01 {
				if (value & 0x01) == 0x01 {
					bus |= 0x80
				} else {
					bus &= 0xFF7F
				}
				ppu.openBusDecayStamp[i] = int32(ppu.Frame)
			} else if ppu.Frame-uint64(ppu.openBusDecayStamp[i]) > 30 {
				bus &= 0xFF7F
			}
			value >>= 1
			mask >>= 1
		}

		ppu.openBus = byte(bus)
	}
}

func (ppu *PPU) SetBusAddress(addr uint16) {
	ppu.ppuBusAddress = addr
	ppu.console.Cartridge.Mapper.NotifyVRAMAddressChange(addr)
}

// XXX: Not Implemented yet
// XXX: !_settings->CheckFlag(EmulationFlags::EnablePpuOamRowCorruption)
func (ppu *PPU) ProcessOAMCorruption() {
	return
}

// XXX: Not Implemented yet
// XXX: !_settings->CheckFlag(EmulationFlags::EnablePpuOamRowCorruption
func (ppu *PPU) SetOAMCorruptionFlags() {
	return
}

// Taken from http://wiki.nesdev.com/w/index.php/The_skinny_on_NES_scrolling#Wrapping_around
func (ppu *PPU) IncHorizontalScrolling() {
	//Increase coarse X scrolling value.
	addr := ppu.state.VideoRAMAddr
	if (addr & 0x001F) == 31 {
		addr = (addr & ^uint16(0x001F)) ^ 0x0400
	} else {
		addr++
	}
	ppu.state.VideoRAMAddr = addr
}

// Taken from http://wiki.nesdev.com/w/index.php/The_skinny_on_NES_scrolling#Wrapping_around
func (ppu *PPU) IncVerticalScrolling() {
	addr := ppu.state.VideoRAMAddr

	if (addr & 0x7000) != 0x7000 {
		// if fine Y < 7
		// increment fine Y
		addr += 0x1000
	} else {
		// fine Y = 0
		addr &= ^uint16(0x7000)
		// let y = coarse Y
		y := (addr & 0x03E0) >> 5
		if y == 29 {
			// coarse Y = 0
			y = 0
			// switch vertical nametable
			addr ^= 0x0800
		} else if y == 31 {
			y = 0
		} else {
			y++
		}
		addr = (addr & ^uint16(0x03E0)) | (y << 5)
	}

	ppu.state.VideoRAMAddr = addr
}

func (ppu *PPU) GetPixelColor() byte {
	offset := byte(ppu.state.XScroll)
	backgroundColor := byte(0)
	spriteBGColor := byte(0)

	if ppu.Cycle > ppu.minimumDrawBGCycle {
		// BackgroundMask = false: Hide background in leftmost 8 pixels of screen
		lo := ((ppu.state.LowBitShift << offset) & 0x8000) >> 15
		hi := ((ppu.state.HighBitShift << offset) & 0x8000) >> 14
		spriteBGColor = byte(lo | hi)
		// XXX: settings->GetbckgroundEnabled
		backgroundColor = spriteBGColor
	}

	if ppu.hasSprite[ppu.Cycle] && ppu.Cycle > ppu.minimumDrawSpriteCycle {
		for i := byte(0); i < byte(ppu.spriteCount); i++ {
			shift := int32(ppu.Cycle) - int32(ppu.spriteTiles[i].spriteX) - 1
			if shift >= 0 && shift < 8 {
				ppu.lastSprite = &ppu.spriteTiles[i]
				var spriteColor byte
				if ppu.spriteTiles[i].horizontalMirror {
					lo := ((ppu.lastSprite.lowByte >> shift) & 0x01)
					hi := (((ppu.lastSprite.highByte >> shift) & 0x01) << 1)
					spriteColor = lo | hi
				} else {
					lo := ((ppu.lastSprite.lowByte << shift) & 0x80) >> 7
					hi := ((ppu.lastSprite.highByte << shift) & 0x80) >> 6
					spriteColor = lo | hi
				}

				if spriteColor != 0 {
					if i == 0 && spriteBGColor != 0 && ppu.sprite0Visible && ppu.Cycle != 256 && ppu.flags.backgroundEnabled && ppu.statusFlags.Sprite0Hit == false && ppu.Cycle > ppu.minimumDrawSpriteStandardCycle {
						ppu.statusFlags.Sprite0Hit = true
					}

					// XXX: Settings->GetSpritesEnabled()
					if backgroundColor == 0 || ppu.spriteTiles[i].backgroundPriority == false {
						return byte(ppu.lastSprite.paletteOffset) + spriteColor
					}
				}
			}
		}
	}

	if (offset + byte(uint16(ppu.Cycle-1)&0x07)) < 8 {
		return byte(ppu.previousTile.paletteOffset) + backgroundColor
	}

	return byte(ppu.currentTile.paletteOffset) + backgroundColor
}

func (ppu *PPU) ShiftTileRegisters() {
	ppu.state.LowBitShift <<= 1
	ppu.state.HighBitShift <<= 1
}

func (ppu *PPU) ProcessSpriteEvaluation() {
	// XXX: PAL
	if ppu.IsRenderingEnabled() {
		if ppu.Cycle < 65 {
			ppu.oamCopyBuffer = 0xFF
			ppu.secondarySpriteRAM[(ppu.Cycle-1)>>1] = 0xFF
		} else {
			if ppu.Cycle == 65 {
				ppu.sprite0Added = false
				ppu.spriteInRange = false
				ppu.secondaryOAMAddr = 0

				ppu.overflowBugCounter = 0

				ppu.oamCopyDone = false
				ppu.spriteAddrH = byte(ppu.state.SpriteRAMAddr>>2) & 0x3F
				ppu.spriteAddrL = byte(ppu.state.SpriteRAMAddr) & 0x03

				ppu.firstVisibleSpriteAddr = ppu.spriteAddrH * 4
				ppu.lastVisibleSpriteAddr = ppu.firstVisibleSpriteAddr
			} else if ppu.Cycle == 256 {
				ppu.sprite0Visible = ppu.sprite0Added
				ppu.spriteCount = (ppu.secondaryOAMAddr >> 2)
			}

			if (ppu.Cycle & 0x01) == 0x01 {
				ppu.oamCopyBuffer = ppu.ReadSpriteRAM(byte(ppu.state.SpriteRAMAddr))
			} else {
				if ppu.oamCopyDone {
					ppu.spriteAddrH = (ppu.spriteAddrH + 1) & 0x3F
					if ppu.secondaryOAMAddr >= 0x20 {
						ppu.oamCopyBuffer = ppu.secondarySpriteRAM[ppu.secondaryOAMAddr&0x1F]
					}
				} else {
					var spriteSize int
					if ppu.flags.largeSprites {
						spriteSize = 16
					} else {
						spriteSize = 8
					}
					if ppu.spriteInRange == false && ppu.ScanLine >= int(ppu.oamCopyBuffer) && ppu.ScanLine < (int(ppu.oamCopyBuffer)+spriteSize) {
						ppu.spriteInRange = true
					}

					if ppu.secondaryOAMAddr < 0x20 {
						ppu.secondarySpriteRAM[ppu.secondaryOAMAddr] = ppu.oamCopyBuffer

						if ppu.spriteInRange {
							ppu.spriteAddrL++
							ppu.secondaryOAMAddr++

							if ppu.spriteAddrH == 0 {
								ppu.sprite0Added = true
							}

							if (ppu.secondaryOAMAddr & 0x03) == 0 {
								ppu.spriteInRange = false
								ppu.spriteAddrL = 0
								ppu.lastVisibleSpriteAddr = ppu.spriteAddrH * 4
								ppu.spriteAddrH = (ppu.spriteAddrH + 1) & 0x3F
								if ppu.spriteAddrH == 0 {
									ppu.oamCopyDone = true
								}
							}
						} else {
							ppu.spriteAddrH = (ppu.spriteAddrH + 1) & 0x3F
							if ppu.spriteAddrH == 0 {
								ppu.oamCopyDone = true
							}
						}
					} else {
						ppu.oamCopyBuffer = ppu.secondarySpriteRAM[ppu.secondaryOAMAddr&0x1F]

						if ppu.spriteInRange {
							ppu.statusFlags.SpriteOverflow = true
							ppu.spriteAddrL = (ppu.spriteAddrL + 1)
							if ppu.spriteAddrL == 4 {
								ppu.spriteAddrH = (ppu.spriteAddrH + 1) & 0x3F
								ppu.spriteAddrL = 0
							}

							if ppu.overflowBugCounter == 0 {
								ppu.overflowBugCounter = 3
							} else if ppu.overflowBugCounter > 0 {
								ppu.overflowBugCounter--
								if ppu.overflowBugCounter == 0 {
									ppu.oamCopyDone = true
									ppu.spriteAddrL = 0
								}
							}
						} else {
							ppu.spriteAddrH = (ppu.spriteAddrH + 1) & 0x3F
							ppu.spriteAddrL = (ppu.spriteAddrL + 1) & 0x03

							if ppu.spriteAddrH == 0 {
								ppu.oamCopyDone = true
							}
						}
					}
				}

				ppu.state.SpriteRAMAddr = uint32(ppu.spriteAddrL&0x03) | uint32(ppu.spriteAddrH<<2)
			}
		}
	}
}

func (ppu *PPU) DrawPixel() {
	x := int(ppu.Cycle - 1)
	y := ppu.ScanLine

	// This is called 3.7 million times per second - needs to be as fast as possible.
	if ppu.IsRenderingEnabled() || (ppu.state.VideoRAMAddr&0x3F00) != 0x3F00 {
		color := ppu.GetPixelColor()
		var palette byte
		if (color & 0x03) > 0 {
			palette = ppu.paletteRAM[color]
		} else {
			palette = ppu.paletteRAM[0]
		}
		c := Palette[palette]
		ppu.back.SetRGBA(x, y, c)
	} else {
		// "If the current VRAM address points in the range $3F00-$3FFF during forced blanking, the color indicated by this palette location will be shown on screen instead of the backdrop color."
		palette := ppu.paletteRAM[ppu.state.VideoRAMAddr&0x1F]
		c := Palette[palette]
		ppu.back.SetRGBA(x, y, c)
	}
}

func (ppu *PPU) ProcessScanLine() {
	if ppu.Cycle <= 256 {
		ppu.LoadTileInfo()

		if ppu.prevRenderingEnabled && (ppu.Cycle&0x07) == 0 {
			ppu.IncHorizontalScrolling()
			if ppu.Cycle == 256 {
				ppu.IncVerticalScrolling()
			}
		}

		if ppu.ScanLine >= 0 {
			ppu.DrawPixel()
			ppu.ShiftTileRegisters()

			ppu.ProcessSpriteEvaluation()
		} else if ppu.Cycle < 9 {
			// Pre-render scanline
			if ppu.Cycle == 1 {
				ppu.statusFlags.VerticalBlank = false
				ppu.console.CPU.ClearNMIFlag()
			}

			// XXX: !_settings->CheckFlag(EmulationFlags::DisableOamAddrBug)
			if ppu.state.SpriteRAMAddr > 0x80 && ppu.IsRenderingEnabled() {
				ppu.WriteSpriteRAM(byte(ppu.Cycle-1), ppu.ReadSpriteRAM(byte((ppu.state.SpriteRAMAddr&0xF8)+(ppu.Cycle-1))))
			}
		}
	} else if ppu.Cycle >= 257 && ppu.Cycle <= 320 {
		if ppu.Cycle == 257 {
			ppu.spriteIndex = 0
			for i := range ppu.hasSprite {
				ppu.hasSprite[i] = false
			}
			if ppu.prevRenderingEnabled {
				ppu.state.VideoRAMAddr = (ppu.state.VideoRAMAddr & ^uint16(0x041F)) | (ppu.state.TmpVideoRAMAddr & 0x041F)
			}
		}

		if ppu.IsRenderingEnabled() {
			ppu.state.SpriteRAMAddr = 0

			if ((ppu.Cycle - 261) % 8) == 0 {
				ppu.LoadSpriteTileInfo()
			} else if ((ppu.Cycle - 257) % 8) == 0 {
				ppu.ReadVRAM(ppu.GetNameTableAddr(), PPURenderingRead)
			} else if ((ppu.Cycle - 259) & 8) == 0 {
				ppu.ReadVRAM(ppu.GetAttributeAddr(), PPURenderingRead)
			}

			if ppu.ScanLine == -1 && ppu.Cycle >= 280 && ppu.Cycle <= 304 {
				ppu.state.VideoRAMAddr = (ppu.state.VideoRAMAddr & ^uint16(0x7BE0)) | (ppu.state.TmpVideoRAMAddr & 0x7BE0)
			}
		}
	} else if ppu.Cycle >= 321 && ppu.Cycle <= 336 {
		if ppu.Cycle == 321 {
			if ppu.IsRenderingEnabled() {
				// XXX: LoadExtraSprites
				// ppu.LoadExtraSprites()
				ppu.oamCopyBuffer = ppu.secondarySpriteRAM[0]
			}
			ppu.LoadTileInfo()
		} else if ppu.prevRenderingEnabled && (ppu.Cycle == 328 || ppu.Cycle == 336) {
			ppu.LoadTileInfo()
			ppu.state.LowBitShift <<= 8
			ppu.state.HighBitShift <<= 8
			ppu.IncHorizontalScrolling()
		} else {
			ppu.LoadTileInfo()
		}
	} else if ppu.Cycle == 337 || ppu.Cycle == 339 {
		if ppu.IsRenderingEnabled() {
			ppu.ReadVRAM(ppu.GetNameTableAddr(), PPURenderingRead)

			// XXX: NesModel NTSC Only
			// XXX: _settings->GetPpuModel() == PpuModel::Ppu2C02
			if ppu.ScanLine == -1 && ppu.Cycle == 339 && (ppu.Frame&0x01) == 0x01 {
				ppu.Cycle = 340
			}
		}
	}
}

func (ppu *PPU) BeginVBLank() {
	ppu.TriggerNMI()
}

func (ppu *PPU) TriggerNMI() {
	if ppu.flags.vblank {
		ppu.console.CPU.SetNMIFlag()
	}
}

// Step executes a single PPU cycle
func (ppu *PPU) Step() {
	if ppu.Cycle > 339 {
		ppu.Cycle = 0
		ppu.ScanLine++
		if ppu.ScanLine > int(ppu.vblankEnd) {
			ppu.lastUpdatedPixel = -1
			ppu.ScanLine = -1

			if ppu.renderingEnabled {
				ppu.ProcessOAMCorruption()
			}

			ppu.UpdateMinimumDrawCycles()
		}

		if ppu.ScanLine == -1 {
			ppu.statusFlags.SpriteOverflow = false
			ppu.statusFlags.Sprite0Hit = false

			ppu.swapBuffer()
		} else if ppu.ScanLine == 240 {
			ppu.SetBusAddress(ppu.state.VideoRAMAddr)

			ppu.Frame++
		}
	} else {
		ppu.Cycle++

		if ppu.ScanLine < 240 {
			ppu.ProcessScanLine()
		} else if ppu.Cycle == 1 && ppu.ScanLine == int(ppu.nmiScanLine) {
			if ppu.preventVBLFlag == false {
				ppu.statusFlags.VerticalBlank = true
				ppu.BeginVBLank()
			}
			ppu.preventVBLFlag = false
		} // ToDO: PAL
	}

	if ppu.needStateUpdate {
		ppu.UpdateState()
	}
}

func (ppu *PPU) UpdateState() {
	ppu.needStateUpdate = false

	if ppu.prevRenderingEnabled != ppu.renderingEnabled {
		ppu.prevRenderingEnabled = ppu.renderingEnabled

		if ppu.ScanLine < 240 {
			if ppu.prevRenderingEnabled {
				ppu.ProcessOAMCorruption()
			} else if ppu.prevRenderingEnabled == false {
				ppu.SetOAMCorruptionFlags()

				ppu.SetBusAddress(ppu.state.VideoRAMAddr & 0x3FFF)

				if ppu.Cycle >= 65 && ppu.Cycle <= 256 {
					ppu.state.SpriteRAMAddr++

					ppu.spriteAddrH = byte((ppu.state.SpriteRAMAddr >> 2) & 0x3F)
					ppu.spriteAddrL = byte(ppu.state.SpriteRAMAddr & 0x03)
				}
			}
		}
	}

	if ppu.renderingEnabled != (ppu.flags.backgroundEnabled || ppu.flags.spritesEnabled) {
		ppu.renderingEnabled = ppu.flags.backgroundEnabled || ppu.flags.spritesEnabled
		ppu.needStateUpdate = true
	}

	if ppu.updateVRAMAddrDelay > 0 {
		ppu.updateVRAMAddrDelay--

		if ppu.updateVRAMAddrDelay == 0 {
			// XXX: EnablePpu2006ScrollGlitch
			// if ppu.ScanLine < 240 && ppu.IsRenderingEnabled() {
			// 	if ppu.Cycle == 257 {
			// 		ppu.state.VideoRAMAddr &= ppu.updateVRAMAddr
			// 	} else if ppu.Cycle > 0 && (ppu.Cycle&0x07) == 0 && (ppu.Cycle <= 256 || ppu.Cycle > 320) {
			// 		ppu.state.VideoRAMAddr = (ppu.updateVRAMAddr & ^uint16(0x041F)) | (ppu.state.VideoRAMAddr & ppu.updateVRAMAddr & 0x041F)
			// 	} else {
			// 		ppu.state.VideoRAMAddr = ppu.updateVRAMAddr
			// 	}
			// } else {
			ppu.state.VideoRAMAddr = ppu.updateVRAMAddr
			// }

			ppu.state.TmpVideoRAMAddr = ppu.state.VideoRAMAddr

			if ppu.ScanLine >= 240 || ppu.IsRenderingEnabled() == false {
				ppu.SetBusAddress(ppu.state.VideoRAMAddr & 0x3FFF)
			}
		} else {
			ppu.needStateUpdate = true
		}
	}

	if ppu.ignoreVRAMRead > 0 {
		ppu.ignoreVRAMRead--
		if ppu.ignoreVRAMRead > 0 {
			ppu.needStateUpdate = true
		}
	}
}

func (ppu *PPU) GetFrameCycle() uint32 {
	return ((uint32(ppu.ScanLine) + 1) * 341) + ppu.Cycle
}

func (ppu *PPU) swapBuffer() {
	ppu.front, ppu.back = ppu.back, ppu.front
}

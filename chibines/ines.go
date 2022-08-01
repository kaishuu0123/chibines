// refs: github.com/fogleman/nes
package toynes

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

const iNESFileMagic = 0x1a53454e

// 16KiB (0x4000)
const PRG_BLOCK_SIZE = 16384

// 8KiB (0x2000)
const CHR_BLOCK_SIZE = 8192

type iNESFileHeader struct {
	Magic    uint32  // iNES magic number
	NumPRG   byte    // number of PRG-ROM banks (16KB each)
	NumCHR   byte    // number of CHR-ROM banks (8KB each)
	Control1 byte    // control bits
	Control2 byte    // control bits
	NumRAM   byte    // PRG-RAM size (x 8KB)
	_        [7]byte // unused padding
}

// LoadNESFile reads an iNES file (.nes) and returns a Cartridge on success.
// http://wiki.nesdev.com/w/index.php/INES
// http://nesdev.com/NESDoc.pdf (page 28)
func LoadNESFile(path string, console *Console) (*Cartridge, error) {
	// open file
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// read file header
	header := iNESFileHeader{}
	if err := binary.Read(file, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	// verify header magic number
	if header.Magic != iNESFileMagic {
		return nil, errors.New("invalid .nes file")
	}

	// mapper ID
	mapper1 := header.Control1 >> 4
	mapper2 := header.Control2 >> 4
	mapperID := mapper1 | mapper2<<4

	// mirroring type
	mirror1 := header.Control1 & 1
	mirror2 := (header.Control1 >> 3) & 1
	mirror := mirror1 | mirror2<<1

	// battery-backed RAM
	battery := (header.Control1 >> 1) & 1

	// read trainer if present (unused)
	if header.Control1&4 == 4 {
		trainer := make([]byte, 512)
		if _, err := io.ReadFull(file, trainer); err != nil {
			return nil, err
		}
	}

	// read prg-rom bank(s)
	prg := make([]byte, int(header.NumPRG)*PRG_BLOCK_SIZE)
	if _, err := io.ReadFull(file, prg); err != nil {
		return nil, err
	}

	// read chr-rom bank(s)
	chr := make([]byte, int(header.NumCHR)*CHR_BLOCK_SIZE)
	if _, err := io.ReadFull(file, chr); err != nil {
		return nil, err
	}

	// provide chr-rom/ram if not in file
	if header.NumCHR == 0 {
		chr = make([]byte, CHR_BLOCK_SIZE)
	}

	cartridge := NewCartridge(
		console, prg, chr, mapperID, mirror,
		battery, path, header.NumPRG, header.NumCHR,
		uint32(header.NumPRG)*PRG_BLOCK_SIZE,
		uint32(header.NumCHR)*CHR_BLOCK_SIZE,
	)
	console.Cartridge = cartridge

	mapper, err := NewMapper(console)
	if err != nil {
		return nil, err
	}
	cartridge.Mapper = mapper

	if battery != 0 {
		cartridge.EEPROM = NewEEPROM(0, path)
		cartridge.EEPROM.Reset()
	}

	// success
	return cartridge, nil
}

// refs: github.com/fogleman/nes
package chibines

import "log"

type Cartridge struct {
	console *Console

	PRG      []byte // PRG-ROM banks
	CHR      []byte // CHR-ROM banks
	MapperID byte   // mapper ID
	Mapper   Mapper
	Mirror   byte    // mirroring mode
	Battery  byte    // battery present
	EEPROM   *EEPROM // Save EEPROM

	// Meta data (from iNES header)
	ROMFilePath string
	NumPRG      byte
	NumCHR      byte
	PRGSize     uint32
	CHRSize     uint32
	PRGMask     uint32
	CHRMask     uint32

	// for NSF Player
	nsfFileInfo *NSFFileInfo
}

func createMask(size uint32) uint32 {
	// returns 1 less than closest fitting power of 2
	// is this number not a power of two or 0?
	if (size & (size - 1)) != 0 {
		// yea, fix it!
		size--
		size |= size >> 1
		size |= size >> 2
		size |= size >> 4
		size |= size >> 8
		size |= size >> 16
		size++
	} else if size == 0 {
		size++
	}

	size--
	return size
}

func NewCartridge(console *Console, prg, chr []byte, mapperID, mirror, battery byte, romFilePath string, numPRG, numCHR byte, prgSize, chrSize uint32) *Cartridge {
	log.Printf("PRG Size: %d\n", numPRG)
	log.Printf("CHR Size: %d\n", numCHR)
	log.Printf("Has Battery: %v\n", battery == 1)
	log.Printf("Mapper ID: %d\n", mapperID)
	log.Printf("Mirroring: %d\n", mirror)

	return &Cartridge{
		console:     console,
		PRG:         prg,
		CHR:         chr,
		MapperID:    mapperID,
		Mapper:      nil,
		Mirror:      mirror,
		Battery:     battery,
		ROMFilePath: romFilePath,
		NumPRG:      numPRG,
		NumCHR:      numCHR,
		PRGSize:     prgSize,
		CHRSize:     chrSize,
		PRGMask:     createMask(prgSize),
		CHRMask:     createMask(chrSize),
	}
}

func (c *Cartridge) HasChrRom() bool {
	return c.NumCHR > 0
}

func (c *Cartridge) Close() {
	if c.Battery != 0 {
		c.EEPROM.Close()
	}
}

func (c *Cartridge) HasBattery() bool {
	return c.Battery == 1
}

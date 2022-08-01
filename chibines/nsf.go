package chibines

import (
	"encoding/binary"
	"os"
)

type NSFFileHeader struct {
	Header          [5]byte
	Version         byte
	TotalSongs      byte
	StartingSong    byte
	LoadAddress     uint16
	InitAddress     uint16
	PlayAddress     uint16
	SongName        [32]byte
	ArtistName      [32]byte
	CopyrightHolder [32]byte
	PlaySpeedNTSC   uint16
	BankSetup       [8]byte
	PlaySpeedPAL    uint16
	Flags           byte
	SoundChips      byte
	Padding         [4]byte

	// NSFe extensions
	// RipperName  [256]byte
	// TrackName   [20000]byte
	// TrackLength [256]int32
	// TrackFade   [256]int32
}

type NSFFileInfo struct {
	*NSFFileHeader

	ROM []byte
}

func ParseNSFFileInfo(path string) (*NSFFileInfo, error) {
	// open file
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// read file header
	header := NSFFileHeader{}
	if err := binary.Read(file, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	// fmt.Printf("TotalSongs: %d\n", header.TotalSongs)
	// fmt.Printf("SongName: %s\n", header.SongName)
	// fmt.Printf("ArtistName: %s\n", header.ArtistName)
	// fmt.Printf("Copyright: %s\n", header.CopyrightHolder)
	// fmt.Printf("BankSetup: %v\n", header.BankSetup)

	data := make([]byte, stat.Size()-0x80)
	if err := binary.Read(file, binary.LittleEndian, data); err != nil {
		return nil, err
	}

	nsfFileInfo := &NSFFileInfo{
		NSFFileHeader: &header,
	}

	if nsfFileInfo.usesBanks() {
		padding := header.LoadAddress & 0x0FFF
		rom := append(make([]byte, padding), data...)
		nsfFileInfo.ROM = rom
	} else {
		rom := make([]byte, 32*1024)
		copy(rom[header.LoadAddress-0x8000:], data)
		nsfFileInfo.ROM = rom
	}

	return nsfFileInfo, nil
}

func LoadNSFFile(path string, console *Console) (*Cartridge, error) {
	nsfFileInfo, err := ParseNSFFileInfo(path)
	if err != nil {
		return nil, err
	}

	chrROM := make([]byte, 8192)

	var mapperID byte = 0
	if nsfFileInfo.usesBanks() {
		mapperID = 31
	}

	romSize := len(nsfFileInfo.ROM)
	numPRG := (romSize / PRG_BLOCK_SIZE)

	cartridge := NewCartridge(
		console, nsfFileInfo.ROM, chrROM, mapperID,
		0x00, 0x00, path,
		byte(numPRG), 1,
		uint32(romSize), 8192,
	)
	console.Cartridge = cartridge
	console.Cartridge.nsfFileInfo = nsfFileInfo

	mapper, err := NewMapper(console)
	if err != nil {
		return nil, err
	}
	cartridge.Mapper = mapper

	return cartridge, nil
}

func (n *NSFFileInfo) usesBanks() bool {
	for i := 0; i < 8; i++ {
		if n.BankSetup[i] != 0 {
			return true
		}
	}
	return false
}

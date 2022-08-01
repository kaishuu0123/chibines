// ORIGINAL
package chibines

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/edsrzf/mmap-go"
)

type EEPROMMode uint32

const (
	Standby EEPROMMode = iota
	Device
	Bank
	Address
	Read
	Write
)

type EEPROMArea uint8

const (
	_             EEPROMArea = iota
	EEPROM_Memory            = 0b1010
	EEPROM_IDPage            = 0b1011
)

type EEPROMLine struct {
	latch bool
	value bool
}

type EEPROM struct {
	file *os.File
	mmap mmap.MMap

	mode EEPROMMode

	counter uint8
	device  uint8
	bank    uint8
	address uint8
	input   byte
	output  uint16

	response    bool
	Acknowledge bool

	clock *EEPROMLine
	data  *EEPROMLine
}

type EEPROMType uint

const (
	_ EEPROMType = iota
	X24C01
	X24C02
)

func GetEEPROMType(crc uint32, mapperNumber byte) EEPROMType {
	switch crc {
	case 0x81a15eb8:
		return X24C02
	default:
		return X24C01
	}
}

func NewEEPROM(eepromType EEPROMType, romFilePath string) *EEPROM {
	var eepromFile *os.File
	var eepromMMap mmap.MMap
	var eepromSize = 256

	romFileName := fileNameWithoutExtension(romFilePath)
	romFileDir := filepath.Dir(filepath.Clean(romFilePath))
	eepromPath := filepath.Join(romFileDir, romFileName+`.eeprom`)

	_, err := os.Stat(eepromPath)
	if err == nil {
		eepromFile, err = os.OpenFile(eepromPath, os.O_RDWR, 0644)
		if err != nil {
			log.Fatal(err)
		}
		eepromMMap, err = mmap.Map(eepromFile, mmap.RDWR, 0)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("EEPROM: file loaded. Path: %s\n", eepromPath)
	} else if os.IsNotExist(err) {
		file, err := os.Create(eepromPath)
		if err != nil {
			log.Fatal("Failed to create output")
		}
		_, err = file.Seek(int64(eepromSize-1), 0)
		if err != nil {
			log.Fatal("Failed to seek")
		}
		_, err = file.Write([]byte{0})
		if err != nil {
			log.Fatal("Write failed")
		}
		err = file.Close()
		if err != nil {
			log.Fatal("Failed to close file")
		}
		log.Printf("EEPROM: file created. Path: %s\n", eepromPath)
	}

	return &EEPROM{
		file:    eepromFile,
		mmap:    eepromMMap,
		counter: 0,
		clock: &EEPROMLine{
			latch: true,
			value: true,
		},
		data: &EEPROMLine{
			latch: true,
			value: true,
		},
	}
}

func (e *EEPROM) Reset() {
	e.mode = Standby
	e.clock.latch = false
	e.clock.value = false
	e.data.latch = false
	e.data.value = false
	e.device = 0b1010 << 4
	e.bank = 0
	e.address = 0
	e.input = 0
	e.output = 0
	e.Acknowledge = false
	e.counter = 0
	e.response = e.Acknowledge
}

func (e *EEPROM) Read() bool {
	if e.mode == Standby {
		return e.data.value
	}
	return e.response
}

func (e *EEPROM) Write() {
	phase := e.mode

	if e.clock.hi() {
		if e.data.fall() {
			e.counter = 0
			e.mode = Device
		} else if e.data.rise() {
			e.counter = 0
			e.mode = Standby
		}
	}

	if e.clock.fall() {
		if e.counter > 8 {
			e.counter = 1
		} else {
			e.counter += 1
		}
	}

	if !e.clock.rise() {
		return
	}

	switch phase {
	case Device:
		if e.counter <= 8 {
			if e.data.value {
				e.device = e.device<<1 | 1
			} else {
				e.device = e.device<<1 | 0
			}
		} else if e.select_device() != e.Acknowledge {
			e.mode = Standby
		} else if e.device&1 == 1 {
			e.mode = Read
			e.response = e.load()
		} else {
			e.mode = Address
			e.response = e.Acknowledge
		}
	case Bank:
		if e.counter <= 8 {
			if e.data.value {
				e.bank = e.bank<<1 | 1
			} else {
				e.bank = e.bank<<1 | 0
			}
		} else {
			e.mode = Address
			e.response = e.Acknowledge
		}
	case Address:
		if e.counter <= 8 {
			if e.data.value {
				e.address = e.address<<1 | 1
			} else {
				e.address = e.address<<1 | 0
			}
		} else {
			e.mode = Write
			e.response = e.Acknowledge
		}
	case Read:
		if e.counter <= 8 {
			e.response = (e.output >> (uint16(8 - e.counter)) & 0x01) == 0x01
		} else if e.data.value == e.Acknowledge {
			e.address += 1
			if e.address == 0 {
				e.bank++
			}
			e.response = e.load()
		} else {
			e.mode = Standby
		}
	case Write:
		if e.counter <= 8 {
			if e.data.value {
				e.input = e.input<<1 | 1
			} else {
				e.input = e.input<<1 | 0
			}
		} else {
			e.response = e.store()
			e.address += 1
			if e.address == 0 {
				e.bank++
			}
		}
	}
}

func (e *EEPROM) SetClock(bit bool) {
	e.clock.latch = e.clock.value
	e.clock.value = bit
}

func (e *EEPROM) SetData(bit bool) {
	e.data.latch = e.data.value
	e.data.value = bit
}

func (e *EEPROM) select_device() bool {
	switch e.device >> 4 {
	case uint8(EEPROM_Memory):
		return e.Acknowledge
	case uint8(EEPROM_IDPage):
		return !e.Acknowledge
	default:
		return !e.Acknowledge
	}
}

func (e *EEPROM) load() bool {
	switch e.device >> 4 {
	case uint8(EEPROM_Memory):
		offset := uint32(e.device>>1)<<8 | uint32(e.address)
		addr := byte(offset & 255)
		e.output = uint16(e.mmap[addr])
		return e.Acknowledge
	case uint8(EEPROM_IDPage):
		return !e.Acknowledge
	default:
		return !e.Acknowledge
	}
}

func (e *EEPROM) store() bool {
	switch e.device >> 4 {
	case uint8(EEPROM_Memory):
		offset := uint32(e.device>>1)<<8 | uint32(e.address)
		addr := byte(offset & 255)
		e.mmap[addr] = e.input
		return e.Acknowledge
	case uint8(EEPROM_IDPage):
		return !e.Acknowledge
	default:
		return !e.Acknowledge
	}
}

func (e *EEPROM) Close() {
	e.mmap.Unmap()
	e.file.Close()
}

func (l *EEPROMLine) lo() bool {
	return !l.latch && !l.value
}

func (l *EEPROMLine) hi() bool {
	return l.latch && l.value
}

func (l *EEPROMLine) fall() bool {
	return l.latch && !l.value
}

func (l *EEPROMLine) rise() bool {
	return !l.latch && l.value
}

func fileNameWithoutExtension(filePath string) string {
	fileName := filepath.Base(filePath)
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}

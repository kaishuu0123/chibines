// refs: github.com/libretro/Mesen
package chibines

import (
	"log"
)

type MMC1RegisterType byte

const (
	MMC1_Reg8000 MMC1RegisterType = iota
	MMC1_RegA000
	MMC1_RegC000
	MMC1_RegE000
)

type Mapper001 struct {
	*MapperBase
	*Cartridge
	console *Console

	writeBuffer byte
	shiftCount  byte

	wramDisable bool
	chrMode     byte
	prgMode     byte
	slotSelect  uint16

	chrReg0 byte
	chrReg1 byte
	prgReg  byte

	lastWriteCycle uint64

	reg8000 byte
	regA000 byte
	regC000 byte
	regE000 byte

	forceWramOn bool
	lastCHRReg  MMC1RegisterType
}

func NewMapper001(cartridge *Cartridge, console *Console) Mapper {
	mapperBase := NewMapperBase(cartridge)
	mapperBase.prgPageSize = 0x4000
	mapperBase.chrPageSize = 0x1000

	m := &Mapper001{
		MapperBase: mapperBase,
		Cartridge:  cartridge,
		console:    console,
		reg8000:    0x0C,
	}

	// XXX:
	if m.HasChrRom() == false {
		m.forceWramOn = true
	}

	m.lastCHRReg = MMC1_RegA000

	m.UpdateState()

	return m
}

func (m *Mapper001) UpdateState() {
	switch m.reg8000 & 0x03 {
	case 0:
		m.SetMirroringType(MIRROR_SINGLE_SCREEN_A)
	case 1:
		m.SetMirroringType(MIRROR_SINGLE_SCREEN_B)
	case 2:
		m.SetMirroringType(MIRROR_VERTICAL)
	case 3:
		m.SetMirroringType(MIRROR_HORIZONTAL)
	}

	m.wramDisable = (m.regE000 & 0x10) == 0x10

	if (m.reg8000 & 0x04) == 0x04 {
		m.slotSelect = 0x8000
	} else {
		m.slotSelect = 0xC000
	}

	if (m.reg8000 & 0x08) == 0x08 {
		m.prgMode = 16
	} else {
		m.prgMode = 32
	}

	if (m.reg8000 & 0x10) == 0x10 {
		m.chrMode = 4
	} else {
		m.chrMode = 8
	}

	m.chrReg0 = m.regA000 & 0x1F
	m.chrReg1 = m.regC000 & 0x1F
	m.prgReg = m.regE000 & 0x0F

	var extraReg byte
	if m.lastCHRReg == MMC1_RegC000 && m.chrMode == 4 {
		extraReg = m.chrReg1
	} else {
		extraReg = m.chrReg0
	}

	var prgBankSelect byte
	if m.prgSize == 0x80000 {
		prgBankSelect = extraReg & 0x10
	}

	var accessType MemoryAccessType
	if m.wramDisable && m.forceWramOn == false {
		accessType = MEMORY_ACCESS_NO_ACCESS
	} else {
		accessType = MEMORY_ACCESS_READ_WRITE
	}
	var memoryType PRGMemoryType
	if m.HasBattery() {
		memoryType = PRG_MEMORY_SAVE_RAM
	} else {
		memoryType = PRG_MEMORY_WORK_RAM
	}

	if m.saveRAMSize+m.workRAMSize > 0x4000 {
		m.SetCPUMemoryMappingByPageNumber(0x6000, 0x7FFF, (uint16(extraReg)>>2)&0x03, memoryType, accessType)
	} else if m.saveRAMSize+m.workRAMSize > 0x2000 {
		if m.saveRAMSize == 0x2000 && m.workRAMSize == 0x2000 {
			var memType PRGMemoryType
			if ((extraReg >> 3) & 0x01) > 0 {
				memType = PRG_MEMORY_WORK_RAM
			} else {
				memType = PRG_MEMORY_SAVE_RAM
			}
			m.SetCPUMemoryMappingByPageNumber(0x6000, 0x7FFF, 0, memType, accessType)
		} else {
			m.SetCPUMemoryMappingByPageNumber(0x6000, 0x7FFF, (uint16(extraReg)>>2)&0x01, memoryType, accessType)
		}
	} else {
		m.SetCPUMemoryMappingByPageNumber(0x6000, 0x7FFF, 0, memoryType, accessType)
	}

	// if(_romInfo.SubMapperID == 5) {
	// 	//SubMapper 5
	// 	//"001: 5 Fixed PRG    SEROM, SHROM, SH1ROM use a fixed 32k PRG ROM with no banking support.
	// 	SelectPrgPage2x(0, 0);
	// } else {
	// 	if(_prgMode == PrgMode::_32k) {
	// 		SelectPrgPage2x(0, (_prgReg & 0xFE) | prgBankSelect);
	// 	} else if(_prgMode == PrgMode::_16k) {
	// 		if(_slotSelect == SlotSelect::x8000) {
	// 			SelectPRGPage(0, _prgReg | prgBankSelect);
	// 			SelectPRGPage(1, 0x0F | prgBankSelect);
	// 		} else if(_slotSelect == SlotSelect::xC000) {
	// 			SelectPRGPage(0, 0 | prgBankSelect);
	// 			SelectPRGPage(1, _prgReg | prgBankSelect);
	// 		}
	// 	}
	// }

	if m.prgMode == 32 {
		m.SelectPRGPage2x(0, uint16(m.prgReg&0xFE)|uint16(prgBankSelect), PRG_MEMORY_PRG_ROM)
	} else if m.prgMode == 16 {
		if m.slotSelect == 0x8000 {
			m.SelectPRGPage(0, uint16(m.prgReg)|uint16(prgBankSelect), PRG_MEMORY_PRG_ROM)
			m.SelectPRGPage(1, uint16(0x0F)|uint16(prgBankSelect), PRG_MEMORY_PRG_ROM)
		} else if m.slotSelect == 0xC000 {
			m.SelectPRGPage(0, uint16(0)|uint16(prgBankSelect), PRG_MEMORY_PRG_ROM)
			m.SelectPRGPage(1, uint16(m.prgReg)|uint16(prgBankSelect), PRG_MEMORY_PRG_ROM)
		}
	}

	if m.chrMode == 8 {
		m.SelectCHRPage(0, uint16(m.chrReg0)&uint16(0x1E), CHR_MEMORY_DEFAULT)
		m.SelectCHRPage(1, (uint16(m.chrReg0)&uint16(0x1E))+1, CHR_MEMORY_DEFAULT)
	} else if m.chrMode == 4 {
		m.SelectCHRPage(0, uint16(m.chrReg0), CHR_MEMORY_DEFAULT)
		m.SelectCHRPage(1, uint16(m.chrReg1), CHR_MEMORY_DEFAULT)
	}
}

func (m *Mapper001) WriteMemory(address uint16, value byte) {
	switch {
	case address >= 0x8000:
		currentCycle := m.console.CPU.cycleCount
		if currentCycle-m.lastWriteCycle >= 2 {
			m.writeRegister(address, value)
		}
		m.lastWriteCycle = m.console.CPU.cycleCount
	case address >= 0x6000:
		m.MapperBase.WriteMemory(address, value)
	default:
		log.Fatalf("Unhandled Mapper001 write at address: 0x%04X", address)
	}
}

func (m *Mapper001) writeRegister(address uint16, value byte) {
	if m.IsBufferFull(value) {
		switch (address & 0x6000) >> 13 {
		case uint16(MMC1_Reg8000):
			m.reg8000 = m.writeBuffer
		case uint16(MMC1_RegA000):
			m.lastCHRReg = MMC1_RegA000
			m.regA000 = m.writeBuffer
		case uint16(MMC1_RegC000):
			m.lastCHRReg = MMC1_RegC000
			m.regC000 = m.writeBuffer
		case uint16(MMC1_RegE000):
			m.regE000 = m.writeBuffer
		}

		m.UpdateState()

		m.ResetBuffer()
	}
}

func (m *Mapper001) ResetBuffer() {
	m.shiftCount = 0
	m.writeBuffer = 0
}

func (m *Mapper001) IsBufferFull(value byte) bool {
	// Has Reset flag
	if (value & 0x80) == 0x80 {
		m.ResetBuffer()
		m.reg8000 |= 0x0C
		m.UpdateState()
		return false
	} else {
		m.writeBuffer >>= 1
		m.writeBuffer |= ((value << 4) & 0x10)

		m.shiftCount++

		return m.shiftCount == 5
	}
}

func (m *Mapper001) Step() {
}

func (m *Mapper001) ExRead(address uint16) byte {
	return 0x00
}

func (m *Mapper001) ExWrite(address uint16, value byte) {
}

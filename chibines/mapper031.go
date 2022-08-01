// refs: github.com/libretro/Mesen
package toynes

import (
	"log"
)

type Mapper031 struct {
	*MapperBase
	*Cartridge

	bankNumSlots [8]int
}

func NewMapper031(cartridge *Cartridge) Mapper {
	mapperBase := NewMapperBase(cartridge)
	mapperBase.prgPageSize = 0x1000
	mapperBase.chrPageSize = 0x2000

	m := &Mapper031{
		MapperBase: mapperBase,
		Cartridge:  cartridge,
	}

	m.bankNumSlots[7] = len(m.PRG)/(4*1024) - 1
	if m.nsfFileInfo != nil {
		if cartridge.nsfFileInfo.usesBanks() {
			for i := uint16(0); i < 8; i++ {
				m.WriteMemory(0x5FF8+i, cartridge.nsfFileInfo.BankSetup[i])
			}
		}
	}
	// m.WriteMemory(0x5FFF, 0xFF)

	mapperBase.SelectCHRPage(0, 0, CHR_MEMORY_DEFAULT)

	return m
}

func (m *Mapper031) ReadMemory(address uint16) byte {
	if address >= 0x6000 && address < 0x8000 {
		// no prg RAM in this mapper
		return 0xFF
	}

	switch {
	case address >= 0x8000:
		slotNum := (address - 0x8000) >> 12
		offset := m.bankNumSlots[slotNum] * (4 * 1024)
		strippedAddr := int(address - 0x8000 - (0x1000 * slotNum))
		realAddr := offset + strippedAddr
		if len(m.PRG) < realAddr {
			return 0xFF
		}
		return m.PRG[realAddr]
		// return m.MapperBase.ReadMemory(address)
	}

	return 0xFF
}

func (m *Mapper031) WriteMemory(address uint16, value byte) {
	switch {
	case address >= 0x8000:
		return
	case address >= 0x6000:
		return
	case address >= 0x5000:
		m.bankNumSlots[address&0x07] = int(value)
		// m.SelectPRGPage(address&0x07, uint16(value), PRG_MEMORY_PRG_ROM)
	default:
		log.Fatalf("Unhandled Mapper031 write at address: 0x%04X", address)
	}
}

func (m *Mapper031) Step() {
}

func (m *Mapper031) ExRead(address uint16) byte {
	return 0x00
}

func (m *Mapper031) ExWrite(address uint16, value byte) {
}

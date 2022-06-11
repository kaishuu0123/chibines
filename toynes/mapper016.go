// refs: github.com/libretro/Mesen
package toynes

import (
	"log"
)

type Mapper016 struct {
	*MapperBase
	*Cartridge

	console       *Console
	prgPage       byte
	prgBankSelect byte
	irqEnable     bool
	irqCounter    uint16
	irqReload     uint16
}

func NewMapper016(cartridge *Cartridge, console *Console) Mapper {
	mapperBase := NewMapperBase(cartridge)
	mapperBase.prgPageSize = 0x4000
	mapperBase.chrPageSize = 0x400

	if mapperBase.GetPRGPageCount() >= 0x20 {
		mapperBase.SelectPRGPage(1, 0x1F, PRG_MEMORY_PRG_ROM)
	} else {
		mapperBase.SelectPRGPage(1, 0x0F, PRG_MEMORY_PRG_ROM)
	}

	return &Mapper016{
		MapperBase: mapperBase,
		Cartridge:  cartridge,
		console:    console,
	}
}

func (m *Mapper016) ReadMemory(address uint16) byte {
	switch {
	case address >= 0x8000:
		return m.MapperBase.ReadMemory(address)
	case address >= 0x6000:
		if m.Cartridge.EEPROM.Read() {
			return 0x10 | (m.console.CPU.bus.openBus & 0xE7)
		} else {
			return 0x00 | (m.console.CPU.bus.openBus & 0xE7)
		}
	default:
		log.Fatalf("Unhandled Mapper016 read at address: 0x%04X", address)
	}

	return 0x00 | (m.console.CPU.bus.openBus & 0xE7)
}

func (m *Mapper016) WriteMemory(address uint16, value byte) {
	switch {
	case address >= 0x8000:
		switch address & 0x000F {
		case 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07:
			m.SelectCHRPage(address&0x07, uint16(value), CHR_MEMORY_DEFAULT)
		case 0x08:
			if m.GetPRGPageCount() >= 0x20 {
				m.prgPage = value & 0x1F
			} else {
				m.prgPage = value & 0x0F
			}
			m.SelectPRGPage(0, uint16(m.prgPage)|uint16(m.prgBankSelect), PRG_MEMORY_PRG_ROM)
		case 0x09:
			switch value & 0x03 {
			case 0:
				m.SetMirroringType(MIRROR_VERTICAL)
			case 1:
				m.SetMirroringType(MIRROR_HORIZONTAL)
			case 2:
				m.SetMirroringType(MIRROR_SINGLE_SCREEN_A)
			case 3:
				m.SetMirroringType(MIRROR_SINGLE_SCREEN_B)
			}
		case 0x0A:
			m.irqEnable = (value & 0x01) == 0x01
			m.irqCounter = m.irqReload
			m.console.CPU.ClearIRQSource(IRQ_EXTERNAL)
		case 0x0B:
			m.irqReload = (m.irqReload & 0xFF00) | uint16(value)
			// m.irqCounter = (m.irqCounter & 0xFF00) | uint16(value)
		case 0x0C:
			m.irqReload = (m.irqReload & 0xFF) | (uint16(value) << 8)
			// m.irqCounter = (m.irqCounter & 0xFF00) | uint16(value)
		case 0x0D:
			if m.Cartridge.EEPROM != nil {
				m.Cartridge.EEPROM.SetClock((value & 0x20) == 0x20)
				m.Cartridge.EEPROM.SetData((value & 0x40) == 0x40)
				m.Cartridge.EEPROM.Write()
			}
		}
	case address >= 0x6000:
	default:
		log.Fatalf("Unhandled Mapper016 write at address: 0x%04X", address)
	}
}

func (m *Mapper016) Step() {
	if m.irqEnable {
		if m.irqCounter == 0 {
			m.console.CPU.SetIRQSource(IRQ_EXTERNAL)
		}
		m.irqCounter--
	}
}

func (m *Mapper016) ExRead(address uint16) byte {
	return 0x00
}

func (m *Mapper016) ExWrite(address uint16, value byte) {
}

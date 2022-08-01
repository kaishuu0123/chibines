// refs: github.com/libretro/Mesen
package toynes

type Mapper003 struct {
	*MapperBase
	*Cartridge

	console *Console

	enableCopyProtection bool
}

func NewMapper003(cartridge *Cartridge, console *Console) Mapper {
	mapperBase := NewMapperBase(cartridge)
	mapperBase.prgPageSize = 0x8000
	mapperBase.chrPageSize = 0x2000

	mapperBase.SelectPRGPage(0, 0, PRG_MEMORY_PRG_ROM)
	mapperBase.SelectCHRPage(0, 0, CHR_MEMORY_DEFAULT)

	return &Mapper003{
		MapperBase: mapperBase,
		Cartridge:  cartridge,
	}
}

func (m *Mapper003) WriteMemory(address uint16, value byte) {
	switch {
	case address >= 0x8000:
		m.WriteRegister(address, value)
	default:
		m.MapperBase.WriteMemory(address, value)
	}
}

func (m *Mapper003) WriteRegister(address uint16, value byte) {
	m.SelectCHRPage(0, uint16(value), CHR_MEMORY_DEFAULT)
}

func (m *Mapper003) Step() {
}

func (m *Mapper003) ExRead(address uint16) byte {
	return 0x00
}

func (m *Mapper003) ExWrite(address uint16, value byte) {
}

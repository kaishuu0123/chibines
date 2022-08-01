// refs: github.com/libretro/Mesen
package toynes

type Mapper002 struct {
	*MapperBase
	*Cartridge
}

func NewMapper002(cartridge *Cartridge, console *Console) Mapper {
	mapperBase := NewMapperBase(cartridge)
	mapperBase.prgPageSize = 0x4000
	mapperBase.chrPageSize = 0x2000

	v := -1
	mapperBase.SelectPRGPage(0, 0, PRG_MEMORY_PRG_ROM)
	mapperBase.SelectPRGPage(1, uint16(v), PRG_MEMORY_PRG_ROM)

	mapperBase.SelectCHRPage(0, 0, CHR_MEMORY_DEFAULT)

	return &Mapper002{
		MapperBase: mapperBase,
		Cartridge:  cartridge,
	}
}

func (m *Mapper002) WriteMemory(address uint16, value byte) {
	switch {
	case address >= 0x8000:
		m.WriteRegister(address, value)
	default:
		m.MapperBase.WriteMemory(address, value)
	}
}

func (m *Mapper002) WriteRegister(address uint16, value byte) {
	m.SelectPRGPage(0, uint16(value), PRG_MEMORY_PRG_ROM)
}

func (m *Mapper002) Step() {
}

func (m *Mapper002) ExRead(address uint16) byte {
	return 0x00
}

func (m *Mapper002) ExWrite(address uint16, value byte) {
}

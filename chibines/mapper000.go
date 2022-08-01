// refs: github.com/libretro/Mesen
package chibines

type Mapper000 struct {
	*MapperBase
	*Cartridge
}

func NewMapper000(cartridge *Cartridge) Mapper {
	mapperBase := NewMapperBase(cartridge)
	mapperBase.prgPageSize = 0x4000
	mapperBase.chrPageSize = 0x2000

	mapperBase.SelectPRGPage(0, 0, PRG_MEMORY_PRG_ROM)
	mapperBase.SelectPRGPage(1, 1, PRG_MEMORY_PRG_ROM)

	mapperBase.SelectCHRPage(0, 0, CHR_MEMORY_DEFAULT)

	return &Mapper000{
		MapperBase: mapperBase,
		Cartridge:  cartridge,
	}
}

func (m *Mapper000) Step() {
}

func (m *Mapper000) ExRead(address uint16) byte {
	return 0x00
}

func (m *Mapper000) ExWrite(address uint16, value byte) {
}

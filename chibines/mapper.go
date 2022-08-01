// refs: github.com/fogleman/nes
package toynes

import (
	"fmt"
)

type Mapper interface {
	// Address range: $4100-$5FFF, $6000-$7FFF, $8000-$FFFF
	ReadMemory(address uint16) byte
	// Address range: $4100-$5FFF, $6000-$7FFF, $8000-$FFFF
	WriteMemory(address uint16, value byte)
	// Address range: $0000-$1FFF
	ReadVRAM(address uint16) byte
	// Address range: $0000-$1FFF
	WriteVRAM(address uint16, value byte)
	// Address range: $4018-$40FF
	ExRead(address uint16) byte
	// Address range: $4019-$40FF
	ExWrite(address uint16, value byte)

	NotifyVRAMAddressChange(address uint16)

	Step()
}

func NewMapper(console *Console) (Mapper, error) {
	cartridge := console.Cartridge
	switch cartridge.MapperID {
	case 0:
		return NewMapper000(cartridge), nil
	case 1:
		return NewMapper001(cartridge, console), nil
	case 2:
		return NewMapper002(cartridge, console), nil
	case 3:
		return NewMapper003(cartridge, console), nil
	case 4:
		return NewMapper004(cartridge, console), nil
	case 5:
		return NewMapper005(cartridge, console), nil
	case 16:
		return NewMapper016(cartridge, console), nil
	case 31:
		return NewMapper031(cartridge), nil
	}
	err := fmt.Errorf("Unsupported mapper: %d", cartridge.MapperID)
	return nil, err
}

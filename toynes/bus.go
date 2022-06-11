package toynes

import (
	"log"
)

type Bus struct {
	CPU         *CPU
	PPU         *PPU
	APU         *APU
	Controller1 *Controller
	Controller2 *Controller
	Cartridge   *Cartridge
	WRAM        [2048]byte // 2 KiB
	openBus     byte
}

func NewBus(cpu *CPU, ppu *PPU, apu *APU, controller1 *Controller, controller2 *Controller, cartridge *Cartridge) *Bus {
	return &Bus{
		CPU:         cpu,
		PPU:         ppu,
		APU:         apu,
		Controller1: controller1,
		Controller2: controller2,
		Cartridge:   cartridge,
	}
}

func (b *Bus) ReadMemory(address uint16) byte {
	var value byte

	switch {
	case address < 0x2000:
		// $0000-$1FFF
		value = b.WRAM[address&0x07ff]
	case address < 0x4000:
		// $2000-$3FFF
		switch address & 0x07 {
		case 0x02:
			value = b.PPU.ReadRAM(address)
		case 0x04:
			value = b.PPU.ReadRAM(address)
		case 0x07:
			value = b.PPU.ReadRAM(address)
		default:
			value = b.PPU.ReadRAM(address)
		}
	case address >= 0x4000 && address <= 0x4013:
		// $4000-$4013
		// ToDO: correct APU read status
		value = b.APU.readRegister(address)
	case address == 0x4015:
		value = b.APU.readRegister(address)
	case address == 0x4016:
		value = b.Controller1.Read()
	case address == 0x4017:
		value = b.Controller2.Read()
	case address >= 0x4018 && address < 0x4100:
		// $4018-$40FF
		value = b.Cartridge.Mapper.ExRead(address)
	case address >= 0x4100 && address <= 0x5FFF:
		// $4100-$5FFF
		value = b.Cartridge.Mapper.ReadMemory(address)
	case address < 0x8000:
		// $6000-$7FFF
		value = b.Cartridge.Mapper.ReadMemory(address)
	case address <= 0xFFFF:
		// $8000-$FFFF
		value = b.Cartridge.Mapper.ReadMemory(address)
	default:
		log.Fatalf("Unhandled memory read at address: 0x%04X", address)
	}

	b.openBus = value
	return value
}

func (b *Bus) WriteMemory(address uint16, value byte) {
	switch {
	case address < 0x2000:
		b.WRAM[address&0x07ff] = value
	case address < 0x4000:
		// $2000-$3FFF
		switch address & 0x07 {
		case 0x00:
			b.PPU.WriteRAM(address, value)
		case 0x01:
			b.PPU.WriteRAM(address, value)
		case 0x02:
			b.PPU.WriteRAM(address, value)
		case 0x03:
			b.PPU.WriteRAM(address, value)
		case 0x04:
			b.PPU.WriteRAM(address, value)
		case 0x05:
			b.PPU.WriteRAM(address, value)
		case 0x06:
			b.PPU.WriteRAM(address, value)
		case 0x07:
			b.PPU.WriteRAM(address, value)
		default:
			log.Fatalf("Unknown address: %X\n", address)
		}
	case address < 0x4014:
		// $4000-$4013
		b.APU.writeRegister(address, value)
	case address == 0x4014:
		// $4014
		b.PPU.WriteRAM(address, value)
	case address == 0x4015:
		// $4015
		b.APU.writeRegister(address, value)
	case address == 0x4016:
		// $4016
		b.Controller1.Write(value)
		b.Controller2.Write(value)
	case address == 0x4017:
		// $4017
		b.APU.writeRegister(address, value)
	case address == 0x4018:
		// $4018
		//   NOTHING DONE
	case address < 0x4100:
		// $4019-$40FF
		b.Cartridge.Mapper.ExWrite(address, value)
	case address >= 0x4100 && address <= 0x5FFF:
		// $4100-$5FFF
		b.Cartridge.Mapper.WriteMemory(address, value)
	case address < 0x8000:
		// $6000-$7FFF
		b.Cartridge.Mapper.WriteMemory(address, value)
	case address <= 0xFFFF:
		// $8000-$FFFF
		b.Cartridge.Mapper.WriteMemory(address, value)
	default:
		log.Fatalf("Unhandled memory write at address: 0x%04X", address)
	}
}

func (b *Bus) ReadMemory16(address uint16) uint16 {
	lo := uint16(b.ReadMemory(address))
	hi := uint16(b.ReadMemory(address + 1))
	return hi<<8 | lo
}

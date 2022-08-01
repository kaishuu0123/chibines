// refs: github.com/libretro/Mesen
package chibines

import (
	"log"
)

type A12StateChange byte

const (
	A12_STATE_CHANGE_NONE A12StateChange = 0
	A12_STATE_CHANGE_RISE                = 1
	A12_STATE_CHANGE_FALL                = 2
)

type A12Watcher struct {
	lastCycle  uint32
	cyclesDown uint32
}

func (a *A12Watcher) UpdateVRAMAddress(addr uint16, frameCycle uint32) A12StateChange {
	result := A12_STATE_CHANGE_NONE

	if a.cyclesDown > 0 {
		if a.lastCycle > frameCycle {
			a.cyclesDown += (89342 - a.lastCycle) + frameCycle
		} else {
			a.cyclesDown += (frameCycle - a.lastCycle)
		}
	}

	if (addr & 0x1000) == 0 {
		if a.cyclesDown == 0 {
			a.cyclesDown = 1
			result = A12_STATE_CHANGE_FALL
		}
	} else if (addr & 0x1000) == 0x1000 {
		if a.cyclesDown > 10 {
			result = A12_STATE_CHANGE_RISE
		}
		a.cyclesDown = 0
	}
	a.lastCycle = frameCycle

	return result
}

type MMC3RegisterType uint16

const (
	MMC3_Reg8000 MMC3RegisterType = 0x8000
	MMC3_Reg8001 MMC3RegisterType = 0x8001
	MMC3_RegA000 MMC3RegisterType = 0xA000
	MMC3_RegA001 MMC3RegisterType = 0xA001
	MMC3_RegC000 MMC3RegisterType = 0xC000
	MMC3_RegC001 MMC3RegisterType = 0xC001
	MMC3_RegE000 MMC3RegisterType = 0xE000
	MMC3_RegE001 MMC3RegisterType = 0xE001
)

type MMC3State struct {
	Reg8000 byte
	RegA000 byte
	RegA001 byte
}

type Mapper004 struct {
	*MapperBase
	*Cartridge
	console *Console

	currentRegister    uint8
	wramEnabled        bool
	wramWriteProtected bool

	a12Watcher *A12Watcher

	forceMMC3RevAIRQs bool

	state *MMC3State

	irqReloadValue byte
	irqCounter     byte
	irqReload      bool
	irqEnabled     bool
	prgMode        byte
	chrMode        byte
	registers      [8]byte
}

func NewMapper004(cartridge *Cartridge, console *Console) Mapper {
	mapperBase := NewMapperBase(cartridge)
	mapperBase.prgPageSize = 0x2000
	mapperBase.chrPageSize = 0x0400

	m := &Mapper004{
		MapperBase: mapperBase,
		Cartridge:  cartridge,
		console:    console,
		a12Watcher: &A12Watcher{},
		state:      &MMC3State{},
	}

	var memoryType PRGMemoryType
	if m.HasBattery() {
		memoryType = PRG_MEMORY_SAVE_RAM
	} else {
		memoryType = PRG_MEMORY_WORK_RAM
	}
	m.ResetMMC3()
	m.SetCPUMemoryMappingByPageNumber(0x6000, 0x7FFF, 0, memoryType, MEMORY_ACCESS_UNSPECIFIED)
	m.UpdateState()
	m.UpdateMirroring()

	return m
}

func (m *Mapper004) ResetMMC3() {
	m.state.Reg8000 = 0
	m.state.RegA000 = 0
	m.state.RegA001 = 0

	m.chrMode = 0
	m.prgMode = 0

	m.currentRegister = 0

	m.registers[0] = 0
	m.registers[1] = 2
	m.registers[2] = 4
	m.registers[3] = 5
	m.registers[4] = 6
	m.registers[5] = 7
	m.registers[6] = 0
	m.registers[7] = 1

	m.irqCounter = 0
	m.irqReloadValue = 0
	m.irqReload = false
	m.irqEnabled = false

	m.wramEnabled = false
	m.wramWriteProtected = false
}

func (m *Mapper004) UpdateMirroring() {
	if m.GetMirroringType() != MIRROR_FOUR_SCREEN {
		if (m.state.RegA000 & 0x01) == 0x01 {
			m.SetMirroringType(MIRROR_HORIZONTAL)
		} else {
			m.SetMirroringType(MIRROR_VERTICAL)
		}
	}
}

func (m *Mapper004) UpdateCHRMapping() {
	if m.chrMode == 0 {
		m.SelectCHRPage(0, uint16(m.registers[0])&0xFE, CHR_MEMORY_DEFAULT)
		m.SelectCHRPage(1, uint16(m.registers[0])|0x01, CHR_MEMORY_DEFAULT)
		m.SelectCHRPage(2, uint16(m.registers[1])&0xFE, CHR_MEMORY_DEFAULT)
		m.SelectCHRPage(3, uint16(m.registers[1])|0x01, CHR_MEMORY_DEFAULT)

		m.SelectCHRPage(4, uint16(m.registers[2]), CHR_MEMORY_DEFAULT)
		m.SelectCHRPage(5, uint16(m.registers[3]), CHR_MEMORY_DEFAULT)
		m.SelectCHRPage(6, uint16(m.registers[4]), CHR_MEMORY_DEFAULT)
		m.SelectCHRPage(7, uint16(m.registers[5]), CHR_MEMORY_DEFAULT)
	} else if m.chrMode == 1 {
		m.SelectCHRPage(0, uint16(m.registers[2]), CHR_MEMORY_DEFAULT)
		m.SelectCHRPage(1, uint16(m.registers[3]), CHR_MEMORY_DEFAULT)
		m.SelectCHRPage(2, uint16(m.registers[4]), CHR_MEMORY_DEFAULT)
		m.SelectCHRPage(3, uint16(m.registers[5]), CHR_MEMORY_DEFAULT)

		m.SelectCHRPage(4, uint16(m.registers[0])&0xFE, CHR_MEMORY_DEFAULT)
		m.SelectCHRPage(5, uint16(m.registers[0])|0x01, CHR_MEMORY_DEFAULT)
		m.SelectCHRPage(6, uint16(m.registers[1])&0xFE, CHR_MEMORY_DEFAULT)
		m.SelectCHRPage(7, uint16(m.registers[1])|0x01, CHR_MEMORY_DEFAULT)
	}
}

func (m *Mapper004) UpdatePRGMapping() {
	if m.prgMode == 0 {
		m.SelectPRGPage(0, uint16(m.registers[6]), PRG_MEMORY_PRG_ROM)
		m.SelectPRGPage(1, uint16(m.registers[7]), PRG_MEMORY_PRG_ROM)
		v := -2
		m.SelectPRGPage(2, uint16(v), PRG_MEMORY_PRG_ROM)
		v2 := -1
		m.SelectPRGPage(3, uint16(v2), PRG_MEMORY_PRG_ROM)
	} else {
		v := -2
		m.SelectPRGPage(0, uint16(v), PRG_MEMORY_PRG_ROM)
		m.SelectPRGPage(1, uint16(m.registers[7]), PRG_MEMORY_PRG_ROM)
		m.SelectPRGPage(2, uint16(m.registers[6]), PRG_MEMORY_PRG_ROM)
		v2 := -1
		m.SelectPRGPage(3, uint16(v2), PRG_MEMORY_PRG_ROM)
	}
}

func (m *Mapper004) CanWriteToWorkRAM() bool {
	return m.wramEnabled && m.wramWriteProtected == false
}

func (m *Mapper004) WriteRegister(addr uint16, value byte) {
	switch addr & 0xE001 {
	case uint16(MMC3_Reg8000):
		m.state.Reg8000 = value
		m.UpdateState()
	case uint16(MMC3_Reg8001):
		if m.currentRegister <= 1 {
			value &= ^byte(0x01)
		}
		m.registers[m.currentRegister] = value
		m.UpdateState()
	case uint16(MMC3_RegA000):
		m.state.RegA000 = value
		m.UpdateMirroring()
	case uint16(MMC3_RegA001):
		m.state.RegA001 = value
		m.UpdateState()
	case uint16(MMC3_RegC000):
		m.irqReloadValue = value
	case uint16(MMC3_RegC001):
		m.irqCounter = 0
		m.irqReload = true
	case uint16(MMC3_RegE000):
		m.irqEnabled = false
		m.console.CPU.ClearIRQSource(IRQ_EXTERNAL)
	case uint16(MMC3_RegE001):
		m.irqEnabled = true
	}
}

func (m *Mapper004) UpdateState() {
	m.currentRegister = m.state.Reg8000 & 0x07
	m.chrMode = (m.state.Reg8000 & 0x80) >> 7
	m.prgMode = (m.state.Reg8000 & 0x40) >> 6

	// if(_romInfo.MapperID == 4 && _romInfo.SubMapperID == 1) {
	// 	//MMC6
	// 	bool wramEnabled = (_state.Reg8000 & 0x20) == 0x20;

	// 	uint8_t firstBankAccess = (_state.RegA001 & 0x10 ? MemoryAccessType::Write : 0) | (_state.RegA001 & 0x20 ? MemoryAccessType::Read : 0);
	// 	uint8_t lastBankAccess = (_state.RegA001 & 0x40 ? MemoryAccessType::Write : 0) | (_state.RegA001 & 0x80 ? MemoryAccessType::Read : 0);
	// 	if(!wramEnabled) {
	// 		firstBankAccess = MemoryAccessType::NoAccess;
	// 		lastBankAccess = MemoryAccessType::NoAccess;
	// 	}

	// 	for(int i = 0; i < 4; i++) {
	// 		SetCpuMemoryMapping(0x7000 + i * 0x400, 0x71FF + i * 0x400, 0, PrgMemoryType::SaveRam, firstBankAccess);
	// 		SetCpuMemoryMapping(0x7200 + i * 0x400, 0x73FF + i * 0x400, 1, PrgMemoryType::SaveRam, lastBankAccess);
	// 	}
	// } else {
	// 	_wramEnabled = (_state.RegA001 & 0x80) == 0x80;
	// 	_wramWriteProtected = (_state.RegA001 & 0x40) == 0x40;

	// 	if(_romInfo.SubMapperID == 0) {
	// 		MemoryAccessType access;
	// 		if(_wramEnabled) {
	// 			access = CanWriteToWorkRam() ? MemoryAccessType::ReadWrite : MemoryAccessType::Read;
	// 		} else {
	// 			access = MemoryAccessType::NoAccess;
	// 		}
	// 		SetCpuMemoryMapping(0x6000, 0x7FFF, 0, HasBattery() ? PrgMemoryType::SaveRam : PrgMemoryType::WorkRam, access);
	// 	}
	// }

	m.wramEnabled = (m.state.RegA001 & 0x80) == 0x80
	m.wramWriteProtected = (m.state.RegA001 & 0x40) == 0x40
	// XXX: SubMapperID 0 only
	var access MemoryAccessType
	if m.wramEnabled {
		if m.CanWriteToWorkRAM() {
			access = MEMORY_ACCESS_READ_WRITE
		} else {
			access = MEMORY_ACCESS_READ
		}
	} else {
		access = MEMORY_ACCESS_NO_ACCESS
	}

	var memoryType PRGMemoryType
	if m.HasBattery() {
		memoryType = PRG_MEMORY_SAVE_RAM
	} else {
		memoryType = PRG_MEMORY_WORK_RAM
	}
	m.SetCPUMemoryMappingByPageNumber(0x6000, 0x7FFF, 0, memoryType, access)

	m.UpdatePRGMapping()
	m.UpdateCHRMapping()
}

func (m *Mapper004) ReadMemory(address uint16) byte {
	switch {
	case address >= 0x8000:
		return m.MapperBase.ReadMemory(address)
	case address >= 0x6000:
		return m.MapperBase.ReadMemory(address)
	default:
		log.Fatalf("Unhandled Mapper004 read at address: 0x%04X", address)
	}

	return 0x00
}

func (m *Mapper004) WriteMemory(address uint16, value byte) {
	switch {
	case address >= 0x8000:
		m.WriteRegister(address, value)
	case address >= 0x6000:
		m.MapperBase.WriteMemory(address, value)
	default:
		log.Fatalf("Unhandled Mapper004 write at address: 0x%04X", address)
	}
}

func (m *Mapper004) Step() {
}

func (m *Mapper004) ExRead(address uint16) byte {
	return 0x00
}

func (m *Mapper004) ExWrite(address uint16, value byte) {
}

func (m *Mapper004) NotifyVRAMAddressChange(address uint16) {
	if m.a12Watcher.UpdateVRAMAddress(address, m.console.PPU.GetFrameCycle()) == A12_STATE_CHANGE_RISE {
		if m.irqCounter == 0 || m.irqReload {
			m.irqCounter = m.irqReloadValue
		} else {
			m.irqCounter--
		}

		// if(ForceMmc3RevAIrqs() || _console->GetSettings()->CheckFlag(EmulationFlags::Mmc3IrqAltBehavior)) {
		// 	//MMC3 Revision A behavior
		// 	if((count > 0 || _irqReload) && _irqCounter == 0 && _irqEnabled) {
		// 		TriggerIrq();
		// 	}
		// } else {
		// 	if(_irqCounter == 0 && _irqEnabled) {
		// 		TriggerIrq();
		// 	}
		// }

		if m.irqCounter == 0 && m.irqEnabled {
			m.console.CPU.SetIRQSource(IRQ_EXTERNAL)
		}

		m.irqReload = false
	}
}

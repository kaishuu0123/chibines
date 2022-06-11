// refs: github.com/libretro/Mesen
package toynes

type Mapper005MemoryHandler struct {
	console *Console
	ppuRegs [8]byte
}

func NewMapper005MemoryHandler(console *Console) *Mapper005MemoryHandler {
	return &Mapper005MemoryHandler{
		console: console,
	}
}

func (mh *Mapper005MemoryHandler) GetReg(addr uint16) byte {
	return mh.ppuRegs[addr&0x07]
}

func (mh *Mapper005MemoryHandler) ReadRAM(addr uint16) byte {
	return 0
}

func (mh *Mapper005MemoryHandler) WriteRAM(addr uint16, value byte) {
	mh.console.PPU.WriteRAM(addr, value)
	mh.ppuRegs[addr&0x07] = value
}

type Mapper005 struct {
	*MapperBase
	*Cartridge

	console *Console

	ExRAMSize       int
	NtWorkRAMIndex  byte
	NtEmptyIndex    byte
	NtFillModeIndex byte

	// audio MMC5Audio (XXX: Todo)
	mapper004Memoryhandler *Mapper005MemoryHandler

	prgRAMProtect1 byte
	prgRAMProtect2 byte

	fillModeTile  byte
	fillModeColor byte

	verticalSplitEnabled       bool
	verticalSplitRightSide     bool
	verticalSplitDelimiterTile byte
	verticalSplitScroll        byte
	verticalSplitBank          byte

	splitinSplitRegion  bool
	splitVerticalScroll uint32
	splitTile           uint32
	splitTileNumber     int32

	multiplierValue1 byte
	multiplierValue2 byte

	nametableMapping byte
	extendedRAMMode  byte

	exAttributeLastNametableFetch uint16
	exAttrLastFetchCounter        int8
	exAttrSelectedCHRBank         byte

	prgMode  byte
	prgBanks [5]byte

	chrMode      byte
	chrUpperBits byte
	chrBanks     [12]uint16
	lastCHRReg   uint16
	prevCHRA     bool

	irqCounterTarget byte
	irqEnabled       bool
	scanlineCounter  byte
	irqPending       bool

	needInFrame     bool
	ppuInFrame      bool
	ppuIdleCounter  byte
	lastPPUReadAddr uint16
	ntReadCounter   byte
}

func NewMapper005(cartridge *Cartridge, console *Console) Mapper {
	mapperBase := NewMapperBase(cartridge)
	mapperBase.prgPageSize = 0x2000
	mapperBase.chrPageSize = 0x400

	m := &Mapper005{
		MapperBase:             mapperBase,
		Cartridge:              cartridge,
		console:                console,
		mapper004Memoryhandler: NewMapper005MemoryHandler(console),
		ExRAMSize:              0x400,
		NtWorkRAMIndex:         4,
		NtEmptyIndex:           2,
		NtFillModeIndex:        3,
	}

	m.SetExtendedRAMMode(0)

	m.WriteRegister(0x5100, 0x03)

	m.WriteRegister(0x5117, 0xFF)

	m.UpdateCHRBanks(true)

	return m
}

func (m *Mapper005) ReadMemory(address uint16) byte {
	switch {
	case address >= 0x5000 && address <= 0x5206:
		return m.ReadRegister(address)
	case address == 0xFFFA, address == 0xFFFB:
		return m.ReadRegister(address)
	}

	return m.MapperBase.ReadMemory(address)
}

func (m *Mapper005) WriteMemory(address uint16, value byte) {
	switch {
	case address >= 0x5000 && address <= 0x5206:
		m.WriteRegister(address, value)
		return
	}

	if address >= 0x5C00 && address <= 0x5FFF && m.extendedRAMMode <= 1 && m.ppuInFrame == false {
		value = 0
	}
	m.MapperBase.WriteMemory(address, value)
}

func (m *Mapper005) Step() {
	// _audio->Clock()

	if m.ppuIdleCounter > 0 {
		m.ppuIdleCounter--
		if m.ppuIdleCounter == 0 {
			m.ppuInFrame = false
			m.UpdateCHRBanks(true)
		}
	}
}

func (m *Mapper005) ExRead(address uint16) byte {
	return 0x00
}

func (m *Mapper005) ExWrite(address uint16, value byte) {
}

func (m *Mapper005) SwitchPrgBank(reg uint16, value byte) {
	m.prgBanks[reg-0x5113] = value
	m.UpdatePrgBanks()
}

func (m *Mapper005) GetCPUBankInfo(reg uint16, bankNumber *byte, memoryType *PRGMemoryType, accessType *MemoryAccessType) {
	*bankNumber = m.prgBanks[reg-0x5113]
	*memoryType = PRG_MEMORY_PRG_ROM
	if ((*bankNumber&0x80) == 0x00 && reg != 0x5117) || reg == 0x5113 {
		*bankNumber &= 0x07
		*accessType = MEMORY_ACCESS_READ
		if m.prgRAMProtect1 == 0x02 && m.prgRAMProtect2 == 0x01 {
			*accessType |= MEMORY_ACCESS_WRITE
		}

		var realWorkRAMSize int32
		if m.cartridge.HasBattery() {
			realWorkRAMSize = int32(m.workRAMSize) - 0
		} else {
			realWorkRAMSize = int32(m.workRAMSize) - int32(m.ExRAMSize)
		}

		var realSaveRAMSize int32
		if m.cartridge.HasBattery() {
			realSaveRAMSize = int32(m.saveRAMSize) - int32(m.ExRAMSize)
		} else {
			realSaveRAMSize = int32(m.workRAMSize) - 0
		}

		// XXX:
		// if(IsNes20() || _romInfo.IsInDatabase) {
		// 	memoryType = PrgMemoryType::WorkRam;
		// 	if(HasBattery() && (bankNumber <= 3 || realSaveRamSize > 0x2000)) {
		// 		memoryType = PrgMemoryType::SaveRam;
		// 	}

		// 	if(realSaveRamSize + realWorkRamSize != 0x4000 && bankNumber >= 4) {
		// 		//When not 2x 8kb (=16kb), banks 4/5/6/7 select the empty socket and return open bus
		// 		accessType = MemoryAccessType::NoAccess;
		// 	}
		// } else {
		// 	memoryType = HasBattery() ? PrgMemoryType::SaveRam : PrgMemoryType::WorkRam;
		// }
		*memoryType = PRG_MEMORY_WORK_RAM
		if m.HasBattery() && (*bankNumber <= 3 || realSaveRAMSize > 0x2000) {
			*memoryType = PRG_MEMORY_SAVE_RAM

			if realSaveRAMSize+realWorkRAMSize != 0x4000 && *bankNumber >= 4 {
				*accessType = MEMORY_ACCESS_NO_ACCESS
			}
		}

		if *memoryType == PRG_MEMORY_WORK_RAM {
			*bankNumber &= byte((realWorkRAMSize / 0x2000) - 1)
			if m.workRAMSize == uint32(m.ExRAMSize) {
				*accessType = MEMORY_ACCESS_NO_ACCESS
			}
		} else if *memoryType == PRG_MEMORY_SAVE_RAM {
			*bankNumber &= byte((realSaveRAMSize / 0x2000) - 1)
			if m.saveRAMSize == uint32(m.ExRAMSize) {
				*accessType = MEMORY_ACCESS_NO_ACCESS
			}
		}
	} else {
		*accessType = MEMORY_ACCESS_READ
		*bankNumber &= 0x7F
	}
}

func (m *Mapper005) UpdatePrgBanks() {
	var value byte
	var memoryType PRGMemoryType
	var accessType MemoryAccessType

	m.GetCPUBankInfo(0x5113, &value, &memoryType, &accessType)
	m.SetCPUMemoryMappingByPageNumber(0x6000, 0x7FFF, uint16(value), memoryType, accessType)

	if m.prgMode == 3 {
		m.GetCPUBankInfo(0x5114, &value, &memoryType, &accessType)
		m.SetCPUMemoryMappingByPageNumber(0x8000, 0x9FFF, uint16(value), memoryType, accessType)
	}

	m.GetCPUBankInfo(0x5115, &value, &memoryType, &accessType)
	if m.prgMode == 1 || m.prgMode == 2 {
		m.SetCPUMemoryMappingByPageNumber(0x8000, 0xBFFF, uint16(value&0xFE), memoryType, accessType)
	} else {
		m.SetCPUMemoryMappingByPageNumber(0x8000, 0xBFFF, uint16(value), memoryType, accessType)
	}

	if m.prgMode == 2 || m.prgMode == 3 {
		m.GetCPUBankInfo(0x5116, &value, &memoryType, &accessType)
		m.SetCPUMemoryMappingByPageNumber(0xC000, 0xDFFF, uint16(value), memoryType, accessType)
	}

	m.GetCPUBankInfo(0x5117, &value, &memoryType, &accessType)
	if m.prgMode == 0 {
		m.SetCPUMemoryMappingByPageNumber(0x8000, 0xFFFF, uint16(value&0x7C), memoryType, accessType)
	} else if m.prgMode == 1 {
		m.SetCPUMemoryMappingByPageNumber(0xC000, 0xFFFF, uint16(value&0x7E), memoryType, accessType)
	} else if m.prgMode == 2 || m.prgMode == 3 {
		m.SetCPUMemoryMappingByPageNumber(0xE000, 0xFFFF, uint16(value&0x7F), memoryType, accessType)
	}
}

func (m *Mapper005) SwitchCHRBank(reg uint16, value byte) {
	newValue := uint16(value) | (uint16(m.chrUpperBits) << 8)
	if newValue != m.chrBanks[reg-0x5120] || m.lastCHRReg != reg {
		m.chrBanks[reg-0x5120] = newValue
		m.lastCHRReg = reg
		m.UpdateCHRBanks(true)
	}
}

func (m *Mapper005) UpdateCHRBanks(forceUpdate bool) {
	largeSprites := (m.mapper004Memoryhandler.GetReg(0x2000) & 0x20) != 0

	if largeSprites == false {
		m.lastCHRReg = 0
	}

	chrA := largeSprites == false || (m.splitTileNumber >= 32 && m.splitTileNumber < 40) || (m.ppuInFrame == false && m.lastCHRReg <= 0x5127)
	if forceUpdate == false && chrA == m.prevCHRA {
		return
	}
	m.prevCHRA = chrA

	if m.chrMode == 0 {
		var page byte
		if chrA {
			page = 0x07
		} else {
			page = 0x0B
		}
		m.SelectCHRPage8x(0, m.chrBanks[page]<<3, CHR_MEMORY_DEFAULT)
	} else if m.chrMode == 1 {
		var page1 byte
		if chrA {
			page1 = 0x03
		} else {
			page1 = 0x0B
		}
		m.SelectCHRPage4x(0, m.chrBanks[page1]<<2, CHR_MEMORY_DEFAULT)
		var page2 byte
		if chrA {
			page2 = 0x07
		} else {
			page2 = 0x0B
		}
		m.SelectCHRPage4x(1, m.chrBanks[page2]<<2, CHR_MEMORY_DEFAULT)
	} else if m.chrMode == 2 {
		var page1 byte
		if chrA {
			page1 = 0x01
		} else {
			page1 = 0x09
		}
		m.SelectCHRPage2x(0, m.chrBanks[page1]<<1, CHR_MEMORY_DEFAULT)
		var page2 byte
		if chrA {
			page2 = 0x03
		} else {
			page2 = 0x0B
		}
		m.SelectCHRPage2x(1, m.chrBanks[page2]<<1, CHR_MEMORY_DEFAULT)
		var page3 byte
		if chrA {
			page3 = 0x05
		} else {
			page3 = 0x09
		}
		m.SelectCHRPage2x(2, m.chrBanks[page3]<<1, CHR_MEMORY_DEFAULT)
		var page4 byte
		if chrA {
			page4 = 0x07
		} else {
			page4 = 0x0B
		}
		m.SelectCHRPage2x(3, m.chrBanks[page4]<<1, CHR_MEMORY_DEFAULT)
	} else if m.chrMode == 3 {
		var page1 byte
		if chrA {
			page1 = 0x00
		} else {
			page1 = 0x08
		}
		m.SelectCHRPage(0, m.chrBanks[page1], CHR_MEMORY_DEFAULT)
		var page2 byte
		if chrA {
			page2 = 0x01
		} else {
			page2 = 0x09
		}
		m.SelectCHRPage(1, m.chrBanks[page2], CHR_MEMORY_DEFAULT)
		var page3 byte
		if chrA {
			page3 = 0x02
		} else {
			page3 = 0x0A
		}
		m.SelectCHRPage(2, m.chrBanks[page3], CHR_MEMORY_DEFAULT)
		var page4 byte
		if chrA {
			page4 = 0x03
		} else {
			page4 = 0x0B
		}
		m.SelectCHRPage(3, m.chrBanks[page4], CHR_MEMORY_DEFAULT)
		var page5 byte
		if chrA {
			page5 = 0x04
		} else {
			page5 = 0x08
		}
		m.SelectCHRPage(4, m.chrBanks[page5], CHR_MEMORY_DEFAULT)
		var page6 byte
		if chrA {
			page6 = 0x05
		} else {
			page6 = 0x09
		}
		m.SelectCHRPage(5, m.chrBanks[page6], CHR_MEMORY_DEFAULT)
		var page7 byte
		if chrA {
			page7 = 0x06
		} else {
			page7 = 0x0A
		}
		m.SelectCHRPage(6, m.chrBanks[page7], CHR_MEMORY_DEFAULT)
		var page8 byte
		if chrA {
			page8 = 0x07
		} else {
			page8 = 0x0B
		}
		m.SelectCHRPage(7, m.chrBanks[page8], CHR_MEMORY_DEFAULT)
	}
}

func (m Mapper005) SetNametableMapping(value byte) {
	m.nametableMapping = value

	var nametables [4]byte = [4]byte{
		0,
		1,
		0,
		m.NtFillModeIndex,
	}
	if m.extendedRAMMode <= 1 {
		nametables[2] = m.NtWorkRAMIndex
	} else {
		nametables[2] = m.NtEmptyIndex
	}

	for i := 0; i < 4; i++ {
		nametableId := nametables[(value>>(i*2))&0x03]
		if nametableId == m.NtWorkRAMIndex {
			if m.cartridge.HasBattery() {
				startAddr := m.saveRAMSize - uint32(m.ExRAMSize)
				endAddr := startAddr + uint32(m.saveRAMPageSize)
				m.SetPPUMemoryMappingBySourceMemory(0x2000+(uint16(i)*0x400), 0x2000+(uint16(i)*0x400)+0x3FF, m.saveRAM[startAddr:endAddr], MEMORY_ACCESS_READ_WRITE)
			} else {
				startAddr := m.workRAMSize - uint32(m.ExRAMSize)
				endAddr := startAddr + uint32(m.workRAMPageSize)
				m.SetPPUMemoryMappingBySourceMemory(0x2000+(uint16(i)*0x400), 0x2000+(uint16(i)*0x400)+0x3FF, m.workRAM[startAddr:endAddr], MEMORY_ACCESS_READ_WRITE)
			}
		}
	}
}

func (m *Mapper005) SetExtendedRAMMode(mode byte) {
	m.extendedRAMMode = mode

	var accessType MemoryAccessType
	if m.extendedRAMMode <= 1 {
		accessType = MEMORY_ACCESS_WRITE
	} else if m.extendedRAMMode == 2 {
		accessType = MEMORY_ACCESS_READ_WRITE
	} else {
		accessType = MEMORY_ACCESS_READ
	}

	if m.cartridge.HasBattery() {
		m.SetCPUMemoryMappingBySourceOffset(0x5C00, 0x5FFF, PRG_MEMORY_SAVE_RAM, m.saveRAMSize-uint32(m.ExRAMSize), accessType)
	} else {
		m.SetCPUMemoryMappingBySourceOffset(0x5C00, 0x5FFF, PRG_MEMORY_WORK_RAM, m.workRAMSize-uint32(m.ExRAMSize), accessType)
	}

	m.SetNametableMapping(m.nametableMapping)
}

func (m *Mapper005) SetFillModeTile(tile byte) {
	m.fillModeTile = tile
	nt := m.GetNameTable(m.NtFillModeIndex)
	for i := 0; i < (32 * 30); i++ {
		nt[i] = tile
	}
}

func (m *Mapper005) SetFillModeColor(color byte) {
	m.fillModeColor = color
	attributeByte := color | (color << 2) | (color << 4) | (color << 6)
	nt := m.GetNameTable(m.NtFillModeIndex)
	for i := 0; i < 64; i++ {
		nt[i+(32*30)] = attributeByte
	}
}

func (m *Mapper005) DetectScanlineStart(address uint16) {
	if address >= 0x2000 && address <= 0x2FFF {
		if m.lastPPUReadAddr == address {
			m.ntReadCounter++
		} else {
			m.ntReadCounter = 0
		}

		if m.ntReadCounter >= 2 {
			if m.ppuInFrame == false && m.needInFrame == false {
				m.needInFrame = true
				m.scanlineCounter = 0
			} else {
				m.scanlineCounter++
				if m.irqCounterTarget == m.scanlineCounter {
					m.irqPending = true
					if m.irqEnabled {
						m.console.CPU.SetIRQSource(IRQ_EXTERNAL)
					}
				}
			}
			m.splitTileNumber = 0
		}
	} else {
		m.ntReadCounter = 0
	}
}

func (m *Mapper005) ReadVRAM(address uint16) byte {
	isNtFetch := address >= 0x2000 && address <= 0x2FFF && (address&0x3FF) < 0x3C0
	if isNtFetch {
		m.splitinSplitRegion = false
		m.splitTileNumber++

		if m.ppuInFrame {
			m.UpdateCHRBanks(false)
		} else if m.needInFrame {
			m.needInFrame = false
			m.ppuInFrame = true
			m.UpdateCHRBanks(false)
		}
	}
	m.DetectScanlineStart(address)

	m.ppuIdleCounter = 3
	m.lastPPUReadAddr = address

	if m.extendedRAMMode <= 1 && m.ppuInFrame {
		if m.verticalSplitEnabled {
			verticalSplitScroll := (m.verticalSplitScroll + m.scanlineCounter) & 240
			if address >= 0x2000 {
				if isNtFetch {
					tileNumber := byte((m.splitTileNumber + 2) % 42)
					if tileNumber <= 32 && (m.verticalSplitRightSide && tileNumber >= m.verticalSplitDelimiterTile) || (m.verticalSplitRightSide == false && tileNumber < m.verticalSplitDelimiterTile) {
						m.splitinSplitRegion = true
						m.splitTile = ((uint32(verticalSplitScroll) & 0xF8) << 2) | uint32(tileNumber)
						return m.MapperBase.ReadVRAM(uint16(0x5C00 + m.splitTile))
					} else {
						m.splitinSplitRegion = false
					}
				} else if m.splitinSplitRegion {
					a := 0x5FC0 | ((m.splitTile & 0x380) >> 4) | ((m.splitTile & 0x1F) >> 2)
					return m.MapperBase.ReadVRAM(uint16(a))
				}
			} else if m.splitinSplitRegion {
				a := (uint16(m.verticalSplitBank)%(uint16(m.MapperBase.GetCHRPageCount())/4))*0x1000 + ((address & ^uint16(0x07)) | ((uint16(verticalSplitScroll) & 0x07) & 0xFFF))
				return m.CHR[a]
			}
		}

		if m.extendedRAMMode == 1 && (m.splitTileNumber < 32 || m.splitTileNumber >= 40) {
			if isNtFetch {
				m.exAttributeLastNametableFetch = address & 0x03FF
				m.exAttrLastFetchCounter = 3
			} else if m.exAttrLastFetchCounter > 0 {
				m.exAttrLastFetchCounter--
				switch m.exAttrLastFetchCounter {
				case 2:
					value := m.MapperBase.ReadVRAM(0x5C00 + m.exAttributeLastNametableFetch)

					m.exAttrSelectedCHRBank = byte(((uint16(value) & 0x3F) | (uint16(m.chrUpperBits) << 6)) % (uint16(m.chrROMSize) / 0x1000))
					palette := (value & 0xC0) >> 6
					return palette | palette<<2 | palette<<4 | palette<<6
				case 1, 0:
					return m.CHR[uint16(m.exAttrSelectedCHRBank)*0x1000+(address&0xFFF)]
				}
			}
		}
	}

	return m.MapperBase.ReadVRAM(address)
}

func (m *Mapper005) WriteRegister(address uint16, value byte) {
	if address >= 0x5113 && address <= 0x5117 {
		m.SwitchPrgBank(address, value)
	} else if address >= 0x5120 && address <= 0x512B {
		m.SwitchCHRBank(address, value)
	} else {
		switch address {
		case 0x5000, 0x5001, 0x5002, 0x5003, 0x5004, 0x5005, 0x5006, 0x5007, 0x5010, 0x5011, 0x5015:
			// _audio->WriteRegister(addr, value)
		case 0x5100:
			m.prgMode = value & 0x03
			m.UpdatePrgBanks()
		case 0x5101:
			m.chrMode = value & 0x03
			m.UpdateCHRBanks(true)
		case 0x5102:
			m.prgRAMProtect1 = value & 0x03
			m.UpdatePrgBanks()
		case 0x5103:
			m.prgRAMProtect2 = value & 0x03
			m.UpdatePrgBanks()
		case 0x5104:
			m.SetExtendedRAMMode(value & 0x03)
		case 0x5105:
			m.SetNametableMapping(value)
		case 0x5106:
			m.SetFillModeTile(value)
		case 0x5107:
			m.SetFillModeColor(value & 0x03)
		case 0x5130:
			m.chrUpperBits = value & 0x03
		case 0x5200:
			m.verticalSplitEnabled = (value & 0x80) == 0x80
			m.verticalSplitRightSide = (value & 0x40) == 0x40
			m.verticalSplitDelimiterTile = (value & 0x1F)
		case 0x5201:
			m.verticalSplitScroll = value
		case 0x5202:
			m.verticalSplitBank = value
		case 0x5203:
			m.irqCounterTarget = value
		case 0x5204:
			m.irqEnabled = (value & 0x80) == 0x80
			if m.irqEnabled == false {
				m.console.CPU.ClearIRQSource(IRQ_EXTERNAL)
			} else if m.irqEnabled && m.irqPending {
				m.console.CPU.SetIRQSource(IRQ_EXTERNAL)
			}
		case 0x5205:
			m.multiplierValue1 = value
		case 0x5206:
			m.multiplierValue2 = value
		}
	}
}

func (m *Mapper005) ReadRegister(address uint16) byte {
	switch address {
	case 0x5010, 0x5015:
		// return _audio->ReadRegister(addr);
	case 0x5204:
		var a byte
		if m.ppuInFrame {
			a = 0x40
		}
		var b byte
		if m.irqPending {
			b = 0x80
		}
		value := a | b
		m.irqPending = false
		m.console.CPU.ClearIRQSource(IRQ_EXTERNAL)
		return value
	case 0x5205:
		return (m.multiplierValue1 * m.multiplierValue2) & 0xFF
	case 0x5206:
		// XXX:
		// return (m.multiplierValue1 * m.multiplierValue2) >> 8
		return 0
	case 0xFFFA, 0xFFFB:
		m.ppuInFrame = false
		m.UpdateCHRBanks(true)
		m.lastPPUReadAddr = 0
		m.scanlineCounter = 0
		m.irqPending = false
		m.console.CPU.ClearIRQSource(IRQ_EXTERNAL)

		// XXX:
		// return DebugReadRAM(addr);
		// simulate open bus
		return byte((address >> 8) & 0xff)
	}

	return m.console.CPU.bus.openBus
}

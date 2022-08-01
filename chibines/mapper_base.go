// refs: github.com/libretro/Mesen
package chibines

import (
	"log"
	"math"
)

type MemoryAccessType int16

const (
	MEMORY_ACCESS_UNSPECIFIED MemoryAccessType = -1
	MEMORY_ACCESS_NO_ACCESS   MemoryAccessType = 0x00
	MEMORY_ACCESS_READ        MemoryAccessType = 0x01
	MEMORY_ACCESS_WRITE       MemoryAccessType = 0x02
	MEMORY_ACCESS_READ_WRITE  MemoryAccessType = 0x03
)

type MirroringType uint16

const (
	MIRROR_HORIZONTAL MirroringType = iota
	MIRROR_VERTICAL
	MIRROR_SINGLE_SCREEN_A
	MIRROR_SINGLE_SCREEN_B
	MIRROR_FOUR_SCREEN
)

type PRGMemoryType byte

const (
	PRG_MEMORY_PRG_ROM PRGMemoryType = iota
	PRG_MEMORY_SAVE_RAM
	PRG_MEMORY_WORK_RAM
)

type PRGBank struct {
	ptr        []byte
	offset     int32
	memoryType PRGMemoryType
	accessType MemoryAccessType
}

func (v *PRGBank) Set(ptr []byte, offset int32, memoryType PRGMemoryType, accessType MemoryAccessType) {
	v.ptr = ptr
	v.offset = offset
	v.memoryType = memoryType
	v.accessType = accessType
}

type CHRMemoryType byte

const (
	CHR_MEMORY_DEFAULT CHRMemoryType = iota
	CHR_MEMORY_CHR_ROM
	CHR_MEMORY_CHR_RAM
	CHR_MEMORY_CHR_NAMETABLE_RAM
)

type CHRBank struct {
	ptr        []byte
	offset     int32
	memoryType CHRMemoryType
	accessType MemoryAccessType
}

func (v *CHRBank) Set(ptr []byte, offset int32, memoryType CHRMemoryType, accessType MemoryAccessType) {
	v.ptr = ptr
	v.offset = offset
	v.memoryType = memoryType
	v.accessType = accessType
}

type MapperBase struct {
	cartridge      *Cartridge
	nameTables     [4 * 0x0400]byte
	nameTableCount uint32
	prgBanks       [0x100]PRGBank
	chrBanks       [0x100]CHRBank
	chrRAM         [0x2000]byte
	workRAM        []byte
	// XXX
	saveRAM       []byte
	hasCHRRAM     bool
	onlyCHRRAM    bool
	mirroringType MirroringType

	prgSize         uint32
	chrROMSize      uint32
	chrRAMSize      uint32
	prgPageSize     uint16
	saveRAMSize     uint32
	saveRAMPageSize uint16
	workRAMSize     uint32
	workRAMPageSize uint16
	chrPageSize     uint16
	chrRAMPageSize  uint16
}

func NewMapperBase(cartridge *Cartridge) *MapperBase {
	m := &MapperBase{
		cartridge: cartridge,
		hasCHRRAM: !cartridge.HasChrRom(),
	}

	m.nameTableCount = 2

	m.prgSize = cartridge.PRGSize
	m.prgPageSize = uint16(cartridge.PRGSize)
	m.chrROMSize = cartridge.CHRSize
	m.chrPageSize = uint16(cartridge.CHRSize)

	if m.hasCHRRAM {
		// XXX: Magic Number (defaultRamSize)
		m.chrRAMSize = 0x2000
		m.chrRAMPageSize = 0x2000
		m.chrROMSize = m.chrRAMSize
		m.chrPageSize = m.chrRAMPageSize
		m.onlyCHRRAM = true
	}

	if cartridge.HasBattery() {
		// XXX: Magic Number (defaultRamSize)
		m.saveRAM = make([]byte, 0x2000)
		m.saveRAMSize = 0x2000
		m.saveRAMPageSize = 0x2000
	} else {
		// XXX: Magic Number (defaultRamSize)
		m.workRAM = make([]byte, 0x2000)
		m.workRAMSize = 0x2000
		m.workRAMPageSize = 0x2000
	}

	// XX: Impl trainer

	switch cartridge.Mirror {
	case 0:
		m.SetMirroringType(MIRROR_HORIZONTAL)
	case 1:
		m.SetMirroringType(MIRROR_VERTICAL)
	}

	return m
}

func (m *MapperBase) SetCPUMemoryMappingBySourceMemory(startAddr uint16, endAddr uint16, source []byte, accessType MemoryAccessType) {
	startAddr >>= 8
	endAddr >>= 8
	for i := startAddr; i <= endAddr; i++ {
		// XXX: fix magic number (0x100 = 256)
		m.prgBanks[i].ptr = source[(i-startAddr)*0x100 : (i-startAddr+1)*0x100]
		if accessType != MEMORY_ACCESS_UNSPECIFIED {
			m.prgBanks[i].accessType = accessType
		} else {
			m.prgBanks[i].accessType = MEMORY_ACCESS_READ
		}
	}
}

func (m *MapperBase) SetCPUMemoryMappingBySourceOffset(startAddr uint16, endAddr uint16, memoryType PRGMemoryType, sourceOffset uint32, accessType MemoryAccessType) {
	var source []byte
	switch memoryType {
	case PRG_MEMORY_SAVE_RAM:
		source = m.saveRAM[:]
	case PRG_MEMORY_WORK_RAM:
		source = m.workRAM[:]
	default:
		source = m.cartridge.PRG[:]
	}

	firstSlot := int(startAddr >> 8)
	slotCount := int((endAddr - startAddr + 1) >> 8)
	for i := 0; i < slotCount; i++ {
		m.prgBanks[firstSlot+i].offset = int32(sourceOffset) + (int32(i) * 256)
		m.prgBanks[firstSlot+i].memoryType = memoryType
		m.prgBanks[firstSlot+i].accessType = accessType
	}

	m.SetCPUMemoryMappingBySourceMemory(startAddr, endAddr, source[sourceOffset:], accessType)
}

func (m *MapperBase) SetCPUMemoryMappingByPageNumber(startAddr uint16, endAddr uint16, pageNumber uint16, memoryType PRGMemoryType, accessType MemoryAccessType) {
	if startAddr > 0xFF00 || endAddr <= startAddr {
		return
	}

	pageCount := uint32(0)
	pageSize := uint32(0)
	defaultAccessType := MEMORY_ACCESS_READ
	switch memoryType {
	case PRG_MEMORY_PRG_ROM:
		pageCount = m.GetPRGPageCount()
		pageSize = uint32(m.prgPageSize)
	case PRG_MEMORY_SAVE_RAM:
		pageSize = uint32(m.saveRAMPageSize)
		if pageSize == 0 {
			return
		}
		pageCount = m.saveRAMSize / uint32(pageSize)
		defaultAccessType |= MEMORY_ACCESS_WRITE
	case PRG_MEMORY_WORK_RAM:
		pageSize = uint32(m.workRAMPageSize)
		if pageSize == 0 {
			return
		}
		pageCount = m.workRAMSize / pageSize
		defaultAccessType |= MEMORY_ACCESS_WRITE
	default:
		log.Fatalln("Invalid parameter")
	}

	if pageCount == 0 {
		return
	}

	wrapPageNumber := func(page *uint16) {
		if *page < 0 {
			*page = uint16(pageCount) + *page
		} else {
			*page = *page % uint16(pageCount)
		}
	}
	wrapPageNumber(&pageNumber)

	if accessType == MEMORY_ACCESS_UNSPECIFIED {
		accessType = defaultAccessType
	}

	if uint16(endAddr-startAddr) >= uint16(pageSize) {
		addr := uint32(startAddr)
		for addr <= (uint32(endAddr) - pageSize + 1) {
			m.SetCPUMemoryMappingBySourceOffset(uint16(addr), uint16(addr)+uint16(pageSize)-1, memoryType, uint32(pageNumber)*pageSize, accessType)
			addr += pageSize
			pageNumber++
			wrapPageNumber(&pageNumber)
		}
	} else {
		m.SetCPUMemoryMappingBySourceOffset(startAddr, endAddr, memoryType, uint32(pageNumber)*pageSize, accessType)
	}
}

func (m *MapperBase) SelectPRGPage2x(slot uint16, page uint16, memoryType PRGMemoryType) {
	m.SelectPRGPage(slot*2, page, memoryType)
	m.SelectPRGPage(slot*2+1, page+1, memoryType)
}

func (m *MapperBase) SelectPRGPage(slot uint16, page uint16, memoryType PRGMemoryType) {
	if m.prgSize < 0x8000 && m.prgPageSize > uint16(m.prgSize) {
		for slot := uint32(0); slot < (0x8000 / m.prgSize); slot++ {
			startAddr := uint16(0x8000 + slot*m.prgSize)
			endAddr := uint16(startAddr + uint16(m.prgSize) - 1)
			m.SetCPUMemoryMappingByPageNumber(startAddr, endAddr, 0, memoryType, MEMORY_ACCESS_UNSPECIFIED)
		}
	} else {
		startAddr := uint16(0x8000 + slot*m.prgPageSize)
		endAddr := uint16(startAddr + m.prgPageSize - 1)
		m.SetCPUMemoryMappingByPageNumber(startAddr, endAddr, page, memoryType, MEMORY_ACCESS_UNSPECIFIED)
	}
}

func (m *MapperBase) SetPPUMemoryMappingBySourceMemory(startAddr uint16, endAddr uint16, sourceMemory []byte, accessType MemoryAccessType) {
	startAddr >>= 8
	endAddr >>= 8
	for i := startAddr; i <= endAddr; i++ {
		// XXX: fix magic number (0x100 = 256)
		m.chrBanks[i].ptr = sourceMemory[(i-startAddr)*0x100 : (i-startAddr+1)*0x100]
		if accessType != MEMORY_ACCESS_UNSPECIFIED {
			m.chrBanks[i].accessType = accessType
		} else {
			m.chrBanks[i].accessType = MEMORY_ACCESS_READ_WRITE
		}
	}
}

func (m *MapperBase) SetPPUMemoryMappingBySourceOffset(startAddr uint16, endAddr uint16, memoryType CHRMemoryType, sourceOffset uint32, accessType MemoryAccessType) {
	var sourceMemory []byte

	switch memoryType {
	case CHR_MEMORY_DEFAULT:
		if m.onlyCHRRAM == false {
			sourceMemory = m.cartridge.CHR[:]
			memoryType = CHR_MEMORY_CHR_ROM
		} else {
			sourceMemory = m.chrRAM[:]
			memoryType = CHR_MEMORY_CHR_RAM
		}
	case CHR_MEMORY_CHR_ROM:
		sourceMemory = m.cartridge.CHR[:]
	case CHR_MEMORY_CHR_RAM:
		sourceMemory = m.chrRAM[:]
	case CHR_MEMORY_CHR_NAMETABLE_RAM:
		// XXX: 0x400 magic number (nameTable pageSize)
		sourceMemory = m.nameTables[:]
	}

	firstSlot := int(startAddr >> 8)
	slotCount := int((endAddr - startAddr + 1) >> 8)
	for i := 0; i < slotCount; i++ {
		m.chrBanks[firstSlot+i].offset = int32(sourceOffset) + (int32(i) * 256)
		m.chrBanks[firstSlot+i].memoryType = memoryType
		m.chrBanks[firstSlot+i].accessType = accessType
	}

	m.SetPPUMemoryMappingBySourceMemory(startAddr, endAddr, sourceMemory[sourceOffset:], accessType)
}

func (m *MapperBase) SetPPUMemoryMappingByPageNumber(startAddr uint16, endAddr uint16, pageNumber uint16, memoryType CHRMemoryType, accessType MemoryAccessType) {
	if startAddr > 0x3F00 || endAddr > 0x3FFF || endAddr <= startAddr {
		return
	}

	pageCount := uint32(0)
	pageSize := uint32(0)
	defaultAccessType := MEMORY_ACCESS_READ
	switch memoryType {
	case CHR_MEMORY_DEFAULT:
		pageSize = uint32(m.chrPageSize)
		if pageSize == 0 {
			return
		}
		pageCount = m.GetCHRPageCount()
		if m.onlyCHRRAM {
			defaultAccessType |= MEMORY_ACCESS_WRITE
		}
	case CHR_MEMORY_CHR_ROM:
		pageSize = uint32(m.chrPageSize)
		if pageSize == 0 {
			return
		}
		pageCount = m.GetCHRPageCount()
	case CHR_MEMORY_CHR_RAM:
		pageSize = uint32(m.chrRAMPageSize)
		if pageSize == 0 {
			return
		}
		pageCount = uint32(len(m.chrRAM)) / uint32(pageSize)
		defaultAccessType |= MEMORY_ACCESS_WRITE
	case CHR_MEMORY_CHR_NAMETABLE_RAM:
		// XXX: Fix Magic number. (NametableSize, NametableCount)
		pageSize = 0x400
		pageCount = 0x10
		defaultAccessType |= MEMORY_ACCESS_WRITE
	}

	if pageCount == 0 {
		return
	}

	pageNumber = pageNumber % uint16(pageCount)

	if (endAddr - startAddr) >= uint16(pageSize) {
		addr := startAddr
		for addr <= (endAddr - uint16(pageSize) + 1) {
			m.SetPPUMemoryMappingBySourceOffset(addr, addr+uint16(pageSize)-1, memoryType, uint32(pageNumber)*pageSize, accessType)
			addr += uint16(pageSize)
			pageNumber = (pageNumber + 1) % uint16(pageCount)
		}
	} else {
		if accessType == MEMORY_ACCESS_UNSPECIFIED {
			m.SetPPUMemoryMappingBySourceOffset(startAddr, endAddr, memoryType, uint32(pageNumber)*pageSize, defaultAccessType)
		} else {
			m.SetPPUMemoryMappingBySourceOffset(startAddr, endAddr, memoryType, uint32(pageNumber)*pageSize, accessType)
		}
	}
}

func (m *MapperBase) SelectCHRPage(slot uint16, page uint16, memoryType CHRMemoryType) {
	pageSize := uint16(0)
	if memoryType == CHR_MEMORY_CHR_NAMETABLE_RAM {
		// XXX: fix magic number
		pageSize = 0x400
	} else {
		if memoryType == CHR_MEMORY_CHR_RAM {
			pageSize = m.chrRAMPageSize
		} else {
			pageSize = m.chrPageSize
		}
	}

	startAddr := uint16(slot * pageSize)
	endAddr := uint16(startAddr + pageSize - 1)
	m.SetPPUMemoryMappingByPageNumber(startAddr, endAddr, page, memoryType, MEMORY_ACCESS_UNSPECIFIED)
}

func (m *MapperBase) SelectCHRPage2x(slot uint16, page uint16, memoryType CHRMemoryType) {
	m.SelectCHRPage(slot*2, page, memoryType)
	m.SelectCHRPage(slot*2+1, page+1, memoryType)
}

func (m *MapperBase) SelectCHRPage4x(slot uint16, page uint16, memoryType CHRMemoryType) {
	m.SelectCHRPage2x(slot*2, page, memoryType)
	m.SelectCHRPage2x(slot*2+1, page+2, memoryType)
}

func (m *MapperBase) SelectCHRPage8x(slot uint16, page uint16, memoryType CHRMemoryType) {
	m.SelectCHRPage4x(slot*2, page, memoryType)
	m.SelectCHRPage4x(slot*2+1, page+4, memoryType)
}

func (m *MapperBase) GetNameTable(nametableIndex byte) []byte {
	if nametableIndex >= byte(m.nameTableCount) {
		return m.nameTables[:]
	}
	m.nameTableCount = uint32(math.Max(float64(m.nameTableCount), float64(nametableIndex+1)))

	start := nametableIndex * byte(m.nameTableCount)
	return m.nameTables[start:]
}

func (m *MapperBase) SetNameTable(index byte, nametableIndex byte) {
	// XXX: FIX magic number. (NametableCount)
	if nametableIndex >= 0x10 {
		return
	}
	m.nameTableCount = uint32(math.Max(float64(m.nameTableCount), float64(nametableIndex+1)))

	m.SetPPUMemoryMappingByPageNumber(0x2000+uint16(index)*0x400, 0x2000+(uint16(index)+1)*0x400-1, uint16(nametableIndex), CHR_MEMORY_CHR_NAMETABLE_RAM, MEMORY_ACCESS_UNSPECIFIED)
	m.SetPPUMemoryMappingByPageNumber(0x3000+uint16(index)*0x400, 0x3000+(uint16(index)+1)*0x400-1, uint16(nametableIndex), CHR_MEMORY_CHR_NAMETABLE_RAM, MEMORY_ACCESS_UNSPECIFIED)
}

func (m *MapperBase) SetNameTables(nametable1Index, nametable2Index, nametable3Index, nametable4Index byte) {
	m.SetNameTable(0, nametable1Index)
	m.SetNameTable(1, nametable2Index)
	m.SetNameTable(2, nametable3Index)
	m.SetNameTable(3, nametable4Index)
}

func (m *MapperBase) GetMirroringType() MirroringType {
	return m.mirroringType
}

func (m *MapperBase) SetMirroringType(mirrorType MirroringType) {
	m.mirroringType = mirrorType
	switch mirrorType {
	case MIRROR_VERTICAL:
		m.SetNameTables(0, 1, 0, 1)
	case MIRROR_HORIZONTAL:
		m.SetNameTables(0, 0, 1, 1)
	case MIRROR_FOUR_SCREEN:
		m.SetNameTables(0, 1, 2, 3)
	case MIRROR_SINGLE_SCREEN_A:
		m.SetNameTables(0, 0, 0, 0)
	case MIRROR_SINGLE_SCREEN_B:
		m.SetNameTables(1, 1, 1, 1)
	}
}

func (m *MapperBase) ReadMemory(address uint16) byte {
	prgBank := m.prgBanks[address>>8]

	if prgBank.ptr != nil && (prgBank.accessType&MEMORY_ACCESS_READ) > 0 {
		return prgBank.ptr[byte(address)]
	}

	// simulate open bus
	return byte((address >> 8) & 0xff)
}

func (m *MapperBase) WriteMemory(address uint16, value byte) {
	prgBank := m.prgBanks[address>>8]
	if prgBank.ptr != nil && (prgBank.accessType&MEMORY_ACCESS_WRITE) > 0 {
		prgBank.ptr[byte(address)] = value
	}
}

func (m *MapperBase) ReadVRAM(address uint16) byte {
	chrBank := m.chrBanks[address>>8]
	if chrBank.ptr != nil && (chrBank.accessType&MEMORY_ACCESS_READ) > 0 {
		return chrBank.ptr[byte(address)]
	}

	// simulate open bus
	return byte((address >> 8) & 0xff)
}

func (m *MapperBase) WriteVRAM(address uint16, value byte) {
	chrBank := m.chrBanks[address>>8]
	if chrBank.ptr != nil && (chrBank.accessType&MEMORY_ACCESS_WRITE) > 0 {
		chrBank.ptr[byte(address)] = value
	}
}

func (m *MapperBase) GetPRGPageCount() uint32 {
	pageSize := m.prgPageSize
	if pageSize > 0 {
		return m.prgSize / uint32(pageSize)
	}

	return 0
}

func (m *MapperBase) GetCHRPageCount() uint32 {
	pageSize := m.chrPageSize
	if pageSize > 0 {
		return m.chrROMSize / uint32(pageSize)
	}
	return 0
}

func (m *MapperBase) NotifyVRAMAddressChange(address uint16) {
	// NOTHING DONE
	// if need, override from mapper
}

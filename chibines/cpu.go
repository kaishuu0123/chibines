// refs: github.com/libretro/Mesen
package toynes

import (
	"fmt"
	"math"
)

const CPUFrequency = 1789773

const (
	NMIVector      uint16 = 0xFFFA
	ResetVector    uint16 = 0xFFFC
	IRQVector      uint16 = 0xFFFE
	ClockRateNtsc  uint32 = 1789773
	ClockRatePal   uint32 = 1662607 // not in use now
	ClockRateDenty uint32 = 1773448 // not in use now
)

type AddressingMode uint16

// addressing modes
const (
	_ AddressingMode = iota
	modeAccumulator
	modeImplied
	modeImmediate
	modeZeroPage
	modeZeroPageX
	modeZeroPageY
	modeRelative
	modeAbsolute
	modeAbsoluteX
	modeAbsoluteXW
	modeAbsoluteY
	modeAbsoluteYW
	modeIndirect
	modeIndirectX
	modeIndirectY
	modeIndirectYW
)

type CPU struct {
	cycleCount        uint64 // number of cycles
	masterClock       uint64
	startClockCount   uint8
	endClockCount     uint8
	needHalt          bool
	spriteDMATransfer bool
	needDummyRead     bool
	spriteDMAOffset   uint8
	cpuWrite          bool
	irqMask           uint8
	currentOperand    uint16
	dmcDMARunning     bool

	state     CPUState
	interrupt byte // interrupt type to perform
	stall     int  // number of cycles to stall
	table     [256]CPUInstruction
	bus       *Bus

	prevRunIRQ  bool
	runIRQ      bool
	prevNMIFlag bool
	prevNeedNMI bool
	needNMI     bool
}

type IRQType byte

const (
	IRQ_EXTERNAL      IRQType = 0x01
	IRQ_FRAME_COUNTER IRQType = 0x02
	IRQ_DMC           IRQType = 0x04
	IRQ_FDS_DISK      IRQType = 0x08
)

type MemoryOperationType byte

const (
	MemoryRead       MemoryOperationType = 0
	MemoryWrite      MemoryOperationType = 1
	ExecuteOpcode    MemoryOperationType = 2
	ExecuteOperand   MemoryOperationType = 3
	PPURenderingRead MemoryOperationType = 4
	DummyRead        MemoryOperationType = 5
	DMCRead          MemoryOperationType = 6
	DummyWrite       MemoryOperationType = 7
)

// stepInfo contains information that the instruction functions use
type stepInfo struct {
	pc   uint16
	mode AddressingMode
}

func NewCPU(console *Console) *CPU {
	cpu := CPU{}
	cpu.createTable()
	return &cpu
}

// Reset resets the CPU to its initial powerup state
func (cpu *CPU) Reset() {
	cpu.irqMask = 0xFF

	cpu.state.PC = cpu.bus.ReadMemory16(ResetVector)
	cpu.state.A = 0
	cpu.state.X = 0
	cpu.state.Y = 0
	cpu.state.SP = 0xFD
	cpu.SetAllFlags(PSFlagsInterrupt)

	// start -1
	cpu.cycleCount = math.MaxUint64
	// XXX: NTSC only
	cpu.startClockCount = 6
	cpu.endClockCount = 6
	// XXX: NTSC only
	cpu.masterClock += 12 + 0

	for i := 0; i < 8; i++ {
		cpu.StartCPUCycle(true)
		cpu.EndCPUCycle(true)
	}
}

// Step executes a single CPU instruction
func (cpu *CPU) Step() int {
	if cpu.stall > 0 {
		cpu.stall--
		return 1
	}

	cycles := cpu.cycleCount

	// prevPC := cpu.state.PC

	opcode := cpu.ReadMemory(cpu.state.PC, ExecuteOpcode)
	cpu.state.PC++
	instruction := cpu.table[opcode]
	mode := instruction.mode
	cpu.currentOperand = cpu.FetchOperand(mode)

	// cpu.PrintInstruction(prevPC, cycles, instruction)

	instruction.fn(cpu.state.PC, mode)

	if cpu.prevRunIRQ || cpu.prevNeedNMI {
		cpu.IRQ()
	}

	return int(cpu.cycleCount - cycles)
}

func (cpu *CPU) StartCPUCycle(forRead bool) {
	if forRead {
		cpu.masterClock += (uint64(cpu.startClockCount) - 1)
	} else {
		cpu.masterClock += (uint64(cpu.startClockCount) + 1)
	}
	cpu.cycleCount++
	// XXX: Magic Number
	for (cpu.bus.PPU.masterClock + 4) <= (cpu.masterClock - 1) {
		cpu.bus.PPU.Step()
		cpu.bus.PPU.masterClock += 4
	}
	// fmt.Printf("AFTER masterClock = %d runTo = %d cycle = %d scanline = %d\n", cpu.bus.PPU.masterClock, (cpu.masterClock - 1), cpu.bus.PPU.Cycle, cpu.bus.PPU.ScanLine)
	cpu.bus.APU.Step()
	cpu.bus.Cartridge.Mapper.Step()
}

func (cpu *CPU) EndCPUCycle(forRead bool) {
	if forRead {
		cpu.masterClock += (uint64(cpu.endClockCount) + 1)
	} else {
		cpu.masterClock += (uint64(cpu.endClockCount) - 1)
	}
	// XXX: Magic Number
	for (cpu.bus.PPU.masterClock + 4) <= (cpu.masterClock - 1) {
		cpu.bus.PPU.Step()
		cpu.bus.PPU.masterClock += 4
	}
	// fmt.Printf("AFTER masterClock = %d runTo = %d cycle = %d scanline = %d\n", cpu.bus.PPU.masterClock, (cpu.masterClock - 1), cpu.bus.PPU.Cycle, cpu.bus.PPU.ScanLine)

	cpu.prevNeedNMI = cpu.needNMI

	if cpu.prevNMIFlag == false && cpu.state.nmiFlag {
		cpu.needNMI = true
	}
	cpu.prevNMIFlag = cpu.state.nmiFlag

	cpu.prevRunIRQ = cpu.runIRQ
	cpu.runIRQ = ((cpu.state.irqFlag & uint32(cpu.irqMask)) > 0) && (cpu.CheckFlag(PSFlagsInterrupt) == false)
}

func (cpu *CPU) ReadMemory(address uint16, opeType MemoryOperationType) byte {
	cpu.ProcessPendingDma(address)

	cpu.StartCPUCycle(true)
	value := cpu.bus.ReadMemory(address)
	cpu.EndCPUCycle(true)

	return value
}

func (cpu *CPU) ReadMemory16(address uint16, opeType MemoryOperationType) uint16 {
	lo := uint16(cpu.ReadMemory(address, opeType))
	hi := uint16(cpu.ReadMemory(address+1, opeType))
	return (hi << 8) | lo
}

func (cpu *CPU) ReadDummy() {
	cpu.ReadMemory(cpu.state.PC, DummyRead)
}

func (cpu *CPU) WriteMemory(address uint16, value byte, opeType MemoryOperationType) {
	cpu.cpuWrite = true
	cpu.StartCPUCycle(false)
	cpu.bus.WriteMemory(address, value)
	cpu.EndCPUCycle(false)
	cpu.cpuWrite = false
}

func (cpu *CPU) ReadMemoryByte() byte {
	value := cpu.ReadMemory(cpu.state.PC, ExecuteOperand)
	cpu.state.PC++
	return value
}

func (cpu *CPU) ReadWord() uint16 {
	value := cpu.ReadMemory16(cpu.state.PC, ExecuteOperand)
	cpu.state.PC += 2
	return value
}

func (cpu *CPU) FetchOperand(mode AddressingMode) uint16 {
	switch mode {
	case modeAccumulator, modeImplied:
		cpu.ReadDummy()
		return 0
	case modeImmediate, modeRelative:
		return cpu.GetImmediate()
	case modeZeroPage:
		return cpu.GetZeroAddr()
	case modeZeroPageX:
		return cpu.GetZeroXAddr()
	case modeZeroPageY:
		return cpu.GetZeroYAddr()
	case modeIndirect:
		return cpu.GetIndirectAddr()
	case modeIndirectX:
		return cpu.GetIndirectXAddr()
	case modeIndirectY:
		return cpu.GetIndirectYAddr(false)
	case modeIndirectYW:
		return cpu.GetIndirectYAddr(true)
	case modeAbsolute:
		return cpu.GetAbsoluteAddr()
	case modeAbsoluteX:
		return cpu.GetAbsoluteAddrX(false)
	case modeAbsoluteXW:
		return cpu.GetAbsoluteAddrX(true)
	case modeAbsoluteY:
		return cpu.GetAbsoluteAddrY(false)
	case modeAbsoluteYW:
		return cpu.GetAbsoluteAddrY(true)
	default:
		return 0
	}

	return 0
}

func (cpu *CPU) GetImmediate() uint16 {
	return uint16(cpu.ReadMemoryByte())
}

func (cpu *CPU) GetZeroAddr() uint16 {
	return uint16(cpu.ReadMemoryByte())
}

func (cpu *CPU) GetZeroXAddr() uint16 {
	value := cpu.ReadMemoryByte()
	cpu.ReadMemory(uint16(value), DummyRead)
	return uint16(value + cpu.state.X)
}

func (cpu *CPU) GetZeroYAddr() uint16 {
	value := cpu.ReadMemoryByte()
	cpu.ReadMemory(uint16(value), DummyRead)
	return uint16(value + cpu.state.Y)
}

func (cpu *CPU) GetIndirectAddr() uint16 {
	return cpu.ReadWord()
}

func (cpu *CPU) GetIndirectXAddr() uint16 {
	zero := cpu.ReadMemoryByte()

	cpu.ReadMemory(uint16(zero), DummyRead)

	zero += cpu.state.X

	var addr uint16
	if zero == 0xFF {
		addr = uint16(cpu.ReadMemory(0x00FF, MemoryRead)) | (uint16(cpu.ReadMemory(0x0000, MemoryRead)) << 8)
	} else {
		addr = cpu.ReadMemory16(uint16(zero), MemoryRead)
	}
	return addr
}

func (cpu *CPU) GetIndirectYAddr(dummyRead bool) uint16 {
	zero := cpu.ReadMemoryByte()

	var addr uint16
	if zero == 0xFF {
		addr = uint16(cpu.ReadMemory(0x00FF, MemoryRead)) | (uint16(cpu.ReadMemory(0x0000, MemoryRead)) << 8)
	} else {
		addr = cpu.ReadMemory16(uint16(zero), MemoryRead)
	}

	pageCrossed := pagesDiffer(addr, uint16(cpu.state.Y))
	if pageCrossed || dummyRead {
		var v uint16
		if pageCrossed {
			v = 0x100
		} else {
			v = 0
		}
		cpu.ReadMemory(addr+uint16(cpu.state.Y)-v, DummyRead)
	}

	return addr + uint16(cpu.state.Y)
}

func (cpu *CPU) GetAbsoluteAddr() uint16 {
	return cpu.ReadWord()
}

func (cpu *CPU) GetAbsoluteAddrX(dummyRead bool) uint16 {
	baseAddr := cpu.ReadWord()
	pageCrossed := pagesDiffer(baseAddr, uint16(cpu.state.X))

	if pageCrossed || dummyRead {
		var v uint16
		if pageCrossed {
			v = 0x100
		} else {
			v = 0
		}
		cpu.ReadMemory(baseAddr+uint16(cpu.state.X)-v, DummyRead)
	}

	return baseAddr + uint16(cpu.state.X)
}

func (cpu *CPU) GetAbsoluteAddrY(dummyRead bool) uint16 {
	baseAddr := cpu.ReadWord()
	pageCrossed := pagesDiffer(baseAddr, uint16(cpu.state.Y))

	if pageCrossed || dummyRead {
		var v uint16
		if pageCrossed {
			v = 0x100
		} else {
			v = 0
		}
		cpu.ReadMemory(baseAddr+uint16(cpu.state.Y)-v, DummyRead)
	}

	return baseAddr + uint16(cpu.state.Y)
}

func (cpu *CPU) RunDMATransfer(offsetValue byte) {
	cpu.spriteDMATransfer = true
	cpu.spriteDMAOffset = offsetValue
	cpu.needHalt = true
}

func (cpu *CPU) StartDMCTransfer() {
	cpu.dmcDMARunning = true
	cpu.needDummyRead = true
	cpu.needHalt = true
}

func (cpu *CPU) ProcessPendingDma(readAddress uint16) {
	if cpu.needHalt == false {
		return
	}

	cpu.StartCPUCycle(true)
	cpu.bus.ReadMemory(readAddress)
	cpu.EndCPUCycle(true)
	cpu.needHalt = false

	spriteDMACounter := uint16(0)
	spriteReadAddr := byte(0)
	readValue := byte(0)
	skipDummyReads := (readAddress == 0x4016 || readAddress == 0x4017)

	processCycle := func() {
		if cpu.needHalt {
			cpu.needHalt = false
		} else if cpu.needDummyRead {
			cpu.needDummyRead = false
		}
		cpu.StartCPUCycle(true)
	}

	for cpu.dmcDMARunning || cpu.spriteDMATransfer {
		getCycle := (cpu.cycleCount & 0x01) == 0

		if getCycle {
			if cpu.dmcDMARunning && cpu.needHalt == false && cpu.needDummyRead == false {
				processCycle()
				readValue = cpu.bus.ReadMemory(cpu.bus.APU.GetDMCReadAddress())
				cpu.EndCPUCycle(true)
				cpu.bus.APU.SetDMCReadBuffer(readValue)
				cpu.dmcDMARunning = false
			} else if cpu.spriteDMATransfer {
				processCycle()
				readValue = cpu.bus.ReadMemory(uint16(cpu.spriteDMAOffset)*0x100 + uint16(spriteReadAddr))
				cpu.EndCPUCycle(true)
				spriteReadAddr++
				spriteDMACounter++
			} else {
				if cpu.needHalt || cpu.needDummyRead {
					panic("ProcessPendingDma")
				}
				processCycle()
				if skipDummyReads == false {
					cpu.bus.ReadMemory(readAddress)
				}
				cpu.EndCPUCycle(true)
			}
		} else {
			if cpu.spriteDMATransfer && (spriteDMACounter&0x01) == 0x01 {
				processCycle()
				cpu.bus.WriteMemory(0x2004, readValue)
				cpu.EndCPUCycle(true)
				spriteDMACounter++
				if spriteDMACounter == 0x200 {
					cpu.spriteDMATransfer = false
				}
			} else {
				processCycle()
				if skipDummyReads == false {
					cpu.bus.ReadMemory(readAddress)
				}
				cpu.EndCPUCycle(true)
			}
		}
	}
}

// IRQ - IRQ Interrupt
func (cpu *CPU) IRQ() {
	cpu.ReadDummy() // fetch opcode (and discard it - $00 (BRK) is forced into the opcode register instead)
	cpu.ReadDummy() // read next instruction byte (actually the same as above, since PC increment is suppressed. Also discarded.)
	cpu.push16(cpu.state.PC)

	if cpu.needNMI {
		cpu.needNMI = false
		cpu.push(cpu.Flags() | PSFlagsReserved)
		cpu.SetFlags(PSFlagsInterrupt)

		cpu.state.PC = cpu.ReadMemory16(NMIVector, MemoryRead)
	} else {
		cpu.push(cpu.Flags() | PSFlagsReserved)
		cpu.SetFlags(PSFlagsInterrupt)

		cpu.state.PC = cpu.ReadMemory16(IRQVector, MemoryRead)
	}
}

func (cpu *CPU) SetNMIFlag() {
	cpu.state.nmiFlag = true
}

func (cpu *CPU) ClearNMIFlag() {
	cpu.state.nmiFlag = false
}

func (cpu *CPU) SetIRQSource(irqType IRQType) {
	cpu.state.irqFlag |= uint32(irqType)
}

func (cpu *CPU) HasIRQSource(irqType IRQType) bool {
	return (cpu.state.irqFlag & uint32(irqType)) > 0
}

func (cpu *CPU) ClearIRQSource(irqType IRQType) {
	cpu.state.irqFlag &= ^uint32(irqType)
}

// pagesDiffer returns true if the two addresses reference different pages
// XXX: rename Method Name
func pagesDiffer(a, b uint16) bool {
	return ((a + b) & 0xFF00) != (a & 0xFF00)
}

// XXX: rename Method Name
func pagesDifferSigned(a uint16, b int8) bool {
	return ((a + uint16(b)) & 0xFF00) != (a & 0xFF00)
}

func (cpu *CPU) BranchRelative(branch bool) {
	offset := int8(cpu.currentOperand)

	if branch {
		if cpu.runIRQ && (cpu.prevRunIRQ == false) {
			cpu.runIRQ = false
		}
		cpu.ReadDummy()

		if pagesDifferSigned(cpu.state.PC, offset) {
			cpu.ReadDummy()
		}

		cpu.state.PC = cpu.state.PC + uint16(offset)
	}
}

// push pushes a byte onto the stack
func (cpu *CPU) push(value byte) {
	cpu.WriteMemory(0x100+uint16(cpu.state.SP), value, MemoryWrite)
	cpu.state.SP--
}

// pull pops a byte from the stack
func (cpu *CPU) pull() byte {
	cpu.state.SP++
	return cpu.ReadMemory(0x100+uint16(cpu.state.SP), MemoryRead)
}

// push16 pushes two bytes onto the stack
func (cpu *CPU) push16(value uint16) {
	hi := byte(value >> 8)
	lo := byte(value & 0xFF)
	cpu.push(hi)
	cpu.push(lo)
}

// pull16 pops two bytes from the stack
func (cpu *CPU) pull16() uint16 {
	lo := uint16(cpu.pull())
	hi := uint16(cpu.pull())
	return hi<<8 | lo
}

// PrintInstruction prints the current CPU state
func (cpu *CPU) PrintInstruction(pc uint16, cycle uint64, instruction CPUInstruction) {
	bytes := instruction.size
	name := instruction.name
	w0 := fmt.Sprintf("%02X", instruction.opcode)
	w1 := fmt.Sprintf("%02X", cpu.currentOperand&0x00FF)
	w2 := fmt.Sprintf("%02X", (cpu.currentOperand&0xFF00)>>8)
	if bytes < 2 {
		w1 = "  "
	}
	if bytes < 3 {
		w2 = "  "
	}

	// ToDO: support disassemble func
	fmt.Printf(
		"%04X  %s %s %s  %s %28s"+
			"A:%02X X:%02X Y:%02X P:%02X SP:%02X CYC:%3d\n",
		pc, w0, w1, w2, name, "",
		cpu.state.A, cpu.state.X, cpu.state.Y, cpu.Flags(), cpu.state.SP, cycle)

	// str := fmt.Sprintf(
	// 	"%04X  %s %s %s  %s %28s"+
	// 		"A:%02X X:%02X Y:%02X P:%02X SP:%02X CYC:%3d\n",
	// 	cpu.state.PC, w0, w1, w2, name, "",
	// 	cpu.A, cpu.X, cpu.Y, cpu.Flags(), cpu.SP, cpu.cycleCount)
	// cpu.DebugInstructions = append(cpu.DebugInstructions, str)
}

func (cpu *CPU) PrintInstruction2(pc uint16, cycle uint64, instruction CPUInstruction) {
	bytes := instruction.size
	w0 := fmt.Sprintf("%02X", instruction.opcode)
	w1 := fmt.Sprintf("%02X", cpu.currentOperand&0x00FF)
	w2 := fmt.Sprintf("%02X", (cpu.currentOperand&0xFF00)>>8)
	if bytes < 2 {
		w1 = "00"
	}
	if bytes < 3 {
		w2 = "00"
	}

	// ToDO: support disassemble func
	fmt.Printf(
		"%04X  %s %s %s"+
			" CYC:%d\n",
		pc, w0, w1, w2,
		cycle)
}

// refs: github.com/libretro/Mesen
package chibines

type CPUState struct {
	PC uint16 // program counter
	SP byte   // stack pointer
	A  byte   // accumulator
	X  byte   // x register
	Y  byte   // y register

	// PS
	C byte // carry flag
	Z byte // zero flag
	I byte // interrupt disable flag
	D byte // decimal mode flag
	B byte // break command flag
	U byte // unused flag
	V byte // overflow flag
	N byte // negative flag

	irqFlag    uint32
	cycleCount uint64
	nmiFlag    bool
}

const (
	PSFlagsCarry     = 0x01
	PSFlagsZero      = 0x02
	PSFlagsInterrupt = 0x04
	PSFlagsDecimal   = 0x08
	PSFlagsBreak     = 0x10
	PSFlagsReserved  = 0x20
	PSFlagsOverflow  = 0x40
	PSFlagsNegative  = 0x80
)

func (cpu *CPU) SetAllFlags(flags byte) {
	cpu.state.C = (flags >> 0) & 1
	cpu.state.Z = (flags >> 1) & 1
	cpu.state.I = (flags >> 2) & 1
	cpu.state.D = (flags >> 3) & 1
	cpu.state.B = (flags >> 4) & 1
	cpu.state.U = (flags >> 5) & 1
	cpu.state.V = (flags >> 6) & 1
	cpu.state.N = (flags >> 7) & 1
}

func (cpu *CPU) SetFlags(flags byte) {
	cpu.SetAllFlags(cpu.Flags() | flags)
}

func (cpu *CPU) ClearFlags(flags byte) {
	cpu.SetAllFlags(cpu.Flags() & ^flags)
}

func (cpu *CPU) CheckFlag(flag byte) bool {
	return (cpu.Flags() & flag) == flag
}

// Flags returns the processor status flags
func (cpu *CPU) Flags() byte {
	var flags byte
	flags |= cpu.state.C << 0
	flags |= cpu.state.Z << 1
	flags |= cpu.state.I << 2
	flags |= cpu.state.D << 3
	flags |= cpu.state.B << 4
	flags |= cpu.state.U << 5
	flags |= cpu.state.V << 6
	flags |= cpu.state.N << 7
	return flags
}

// setZN sets the zero flag and the negative flag
func (cpu *CPU) setZN(value byte) {
	cpu.setZ(value)
	cpu.setN(value)
}

// setZ sets the zero flag if the argument is zero
func (cpu *CPU) setZ(value byte) {
	if value == 0 {
		cpu.state.Z = 1
	} else {
		cpu.state.Z = 0
	}
}

// setN sets the negative flag if the argument is negative (high bit is set)
func (cpu *CPU) setN(value byte) {
	if (value & 0x80) == 0x80 {
		cpu.state.N = 1
	} else {
		cpu.state.N = 0
	}
}

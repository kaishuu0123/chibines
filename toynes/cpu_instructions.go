// refs: github.com/libretro/Mesen
package toynes

import (
	"log"
)

type CPUInstruction struct {
	opcode byte
	// instructionNames indicates the name of each instruction
	name string
	// addressing mode
	mode AddressingMode
	// instructionSizes indicates the size of each instruction in bytes
	size byte
	// instructionCycles indicates the number of cycles used by each instruction, not including conditional cycles
	cycles byte
	// instructionPageCycles indicates the number of cycles used by each instruction when a page is crossed
	pageCycles byte
	// instruction function
	fn func(pc uint16, mode AddressingMode)
}

// createTable builds a function table for each instruction
func (c *CPU) createTable() {
	c.table = [256]CPUInstruction{
		{opcode: 0x00, name: "BRK", mode: modeImplied, size: 1, cycles: 7, pageCycles: 0, fn: c.brk},
		{opcode: 0x01, name: "ORA", mode: modeIndirectX, size: 2, cycles: 6, pageCycles: 0, fn: c.ora},
		{opcode: 0x02, name: "KIL", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.kil},
		{opcode: 0x03, name: "SLO", mode: modeIndirectX, size: 2, cycles: 8, pageCycles: 0, fn: c.slo},
		{opcode: 0x04, name: "NOP", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.nop},
		{opcode: 0x05, name: "ORA", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.ora},
		{opcode: 0x06, name: "ASL", mode: modeZeroPage, size: 2, cycles: 5, pageCycles: 0, fn: c.asl},
		{opcode: 0x07, name: "SLO", mode: modeZeroPage, size: 2, cycles: 5, pageCycles: 0, fn: c.slo},
		{opcode: 0x08, name: "PHP", mode: modeImplied, size: 1, cycles: 3, pageCycles: 0, fn: c.php},
		{opcode: 0x09, name: "ORA", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.ora},
		{opcode: 0x0A, name: "ASL", mode: modeAccumulator, size: 1, cycles: 2, pageCycles: 0, fn: c.asl},
		{opcode: 0x0B, name: "ANC", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.anc},
		{opcode: 0x0C, name: "NOP", mode: modeAbsolute, size: 3, cycles: 4, pageCycles: 0, fn: c.nop},
		{opcode: 0x0D, name: "ORA", mode: modeAbsolute, size: 3, cycles: 4, pageCycles: 0, fn: c.ora},
		{opcode: 0x0E, name: "ASL", mode: modeAbsolute, size: 3, cycles: 6, pageCycles: 0, fn: c.asl},
		{opcode: 0x0F, name: "SLO", mode: modeAbsolute, size: 3, cycles: 6, pageCycles: 0, fn: c.slo},
		{opcode: 0x10, name: "BPL", mode: modeRelative, size: 2, cycles: 2, pageCycles: 1, fn: c.bpl},
		{opcode: 0x11, name: "ORA", mode: modeIndirectY, size: 2, cycles: 5, pageCycles: 1, fn: c.ora},
		{opcode: 0x12, name: "KIL", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.kil},
		{opcode: 0x13, name: "SLO", mode: modeIndirectYW, size: 2, cycles: 8, pageCycles: 0, fn: c.slo},
		{opcode: 0x14, name: "NOP", mode: modeZeroPageX, size: 2, cycles: 4, pageCycles: 0, fn: c.nop},
		{opcode: 0x15, name: "ORA", mode: modeZeroPageX, size: 2, cycles: 4, pageCycles: 0, fn: c.ora},
		{opcode: 0x16, name: "ASL", mode: modeZeroPageX, size: 2, cycles: 6, pageCycles: 0, fn: c.asl},
		{opcode: 0x17, name: "SLO", mode: modeZeroPageX, size: 2, cycles: 6, pageCycles: 0, fn: c.slo},
		{opcode: 0x18, name: "CLC", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.clc},
		{opcode: 0x19, name: "ORA", mode: modeAbsoluteY, size: 3, cycles: 4, pageCycles: 1, fn: c.ora},
		{opcode: 0x1A, name: "NOP", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.nop},
		{opcode: 0x1B, name: "SLO", mode: modeAbsoluteYW, size: 3, cycles: 7, pageCycles: 0, fn: c.slo},
		{opcode: 0x1C, name: "NOP", mode: modeAbsoluteX, size: 3, cycles: 4, pageCycles: 1, fn: c.nop},
		{opcode: 0x1D, name: "ORA", mode: modeAbsoluteX, size: 3, cycles: 4, pageCycles: 1, fn: c.ora},
		{opcode: 0x1E, name: "ASL", mode: modeAbsoluteXW, size: 3, cycles: 7, pageCycles: 0, fn: c.asl},
		{opcode: 0x1F, name: "SLO", mode: modeAbsoluteXW, size: 3, cycles: 7, pageCycles: 0, fn: c.slo},
		{opcode: 0x20, name: "JSR", mode: modeAbsolute, size: 3, cycles: 6, pageCycles: 0, fn: c.jsr},
		{opcode: 0x21, name: "AND", mode: modeIndirectX, size: 2, cycles: 6, pageCycles: 0, fn: c.and},
		{opcode: 0x22, name: "KIL", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.kil},
		{opcode: 0x23, name: "RLA", mode: modeIndirectX, size: 2, cycles: 8, pageCycles: 0, fn: c.rla},
		{opcode: 0x24, name: "BIT", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.bit},
		{opcode: 0x25, name: "AND", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.and},
		{opcode: 0x26, name: "ROL", mode: modeZeroPage, size: 2, cycles: 5, pageCycles: 0, fn: c.rol},
		{opcode: 0x27, name: "RLA", mode: modeZeroPage, size: 2, cycles: 5, pageCycles: 0, fn: c.rla},
		{opcode: 0x28, name: "PLP", mode: modeImplied, size: 1, cycles: 4, pageCycles: 0, fn: c.plp},
		{opcode: 0x29, name: "AND", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.and},
		{opcode: 0x2A, name: "ROL", mode: modeAccumulator, size: 1, cycles: 2, pageCycles: 0, fn: c.rol},
		{opcode: 0x2B, name: "ANC", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.anc},
		{opcode: 0x2C, name: "BIT", mode: modeAbsolute, size: 3, cycles: 4, pageCycles: 0, fn: c.bit},
		{opcode: 0x2D, name: "AND", mode: modeAbsolute, size: 3, cycles: 4, pageCycles: 0, fn: c.and},
		{opcode: 0x2E, name: "ROL", mode: modeAbsolute, size: 3, cycles: 6, pageCycles: 0, fn: c.rol},
		{opcode: 0x2F, name: "RLA", mode: modeAbsolute, size: 3, cycles: 6, pageCycles: 0, fn: c.rla},
		{opcode: 0x30, name: "BMI", mode: modeRelative, size: 2, cycles: 2, pageCycles: 1, fn: c.bmi},
		{opcode: 0x31, name: "AND", mode: modeIndirectY, size: 2, cycles: 5, pageCycles: 1, fn: c.and},
		{opcode: 0x32, name: "KIL", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.kil},
		{opcode: 0x33, name: "RLA", mode: modeIndirectYW, size: 2, cycles: 8, pageCycles: 0, fn: c.rla},
		{opcode: 0x34, name: "NOP", mode: modeZeroPageX, size: 2, cycles: 4, pageCycles: 0, fn: c.nop},
		{opcode: 0x35, name: "AND", mode: modeZeroPageX, size: 2, cycles: 4, pageCycles: 0, fn: c.and},
		{opcode: 0x36, name: "ROL", mode: modeZeroPageX, size: 2, cycles: 6, pageCycles: 0, fn: c.rol},
		{opcode: 0x37, name: "RLA", mode: modeZeroPageX, size: 2, cycles: 6, pageCycles: 0, fn: c.rla},
		{opcode: 0x38, name: "SEC", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.sec},
		{opcode: 0x39, name: "AND", mode: modeAbsoluteY, size: 3, cycles: 4, pageCycles: 1, fn: c.and},
		{opcode: 0x3A, name: "NOP", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.nop},
		{opcode: 0x3B, name: "RLA", mode: modeAbsoluteYW, size: 3, cycles: 7, pageCycles: 0, fn: c.rla},
		{opcode: 0x3C, name: "NOP", mode: modeAbsoluteX, size: 3, cycles: 4, pageCycles: 1, fn: c.nop},
		{opcode: 0x3D, name: "AND", mode: modeAbsoluteX, size: 3, cycles: 4, pageCycles: 1, fn: c.and},
		{opcode: 0x3E, name: "ROL", mode: modeAbsoluteXW, size: 3, cycles: 7, pageCycles: 0, fn: c.rol},
		{opcode: 0x3F, name: "RLA", mode: modeAbsoluteXW, size: 3, cycles: 7, pageCycles: 0, fn: c.rla},
		{opcode: 0x40, name: "RTI", mode: modeImplied, size: 1, cycles: 6, pageCycles: 0, fn: c.rti},
		{opcode: 0x41, name: "EOR", mode: modeIndirectX, size: 2, cycles: 6, pageCycles: 0, fn: c.eor},
		{opcode: 0x42, name: "KIL", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.kil},
		{opcode: 0x43, name: "SRE", mode: modeIndirectX, size: 2, cycles: 8, pageCycles: 0, fn: c.sre},
		{opcode: 0x44, name: "NOP", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.nop},
		{opcode: 0x45, name: "EOR", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.eor},
		{opcode: 0x46, name: "LSR", mode: modeZeroPage, size: 2, cycles: 5, pageCycles: 0, fn: c.lsr},
		{opcode: 0x47, name: "SRE", mode: modeZeroPage, size: 2, cycles: 5, pageCycles: 0, fn: c.sre},
		{opcode: 0x48, name: "PHA", mode: modeImplied, size: 1, cycles: 3, pageCycles: 0, fn: c.pha},
		{opcode: 0x49, name: "EOR", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.eor},
		{opcode: 0x4A, name: "LSR", mode: modeAccumulator, size: 1, cycles: 2, pageCycles: 0, fn: c.lsr},
		{opcode: 0x4B, name: "ALR", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.alr},
		{opcode: 0x4C, name: "JMP", mode: modeAbsolute, size: 3, cycles: 3, pageCycles: 0, fn: c.jmp},
		{opcode: 0x4D, name: "EOR", mode: modeAbsolute, size: 3, cycles: 4, pageCycles: 0, fn: c.eor},
		{opcode: 0x4E, name: "LSR", mode: modeAbsolute, size: 3, cycles: 6, pageCycles: 0, fn: c.lsr},
		{opcode: 0x4F, name: "SRE", mode: modeAbsolute, size: 3, cycles: 6, pageCycles: 0, fn: c.sre},
		{opcode: 0x50, name: "BVC", mode: modeRelative, size: 2, cycles: 2, pageCycles: 1, fn: c.bvc},
		{opcode: 0x51, name: "EOR", mode: modeIndirectY, size: 2, cycles: 5, pageCycles: 1, fn: c.eor},
		{opcode: 0x52, name: "KIL", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.kil},
		{opcode: 0x53, name: "SRE", mode: modeIndirectYW, size: 2, cycles: 8, pageCycles: 0, fn: c.sre},
		{opcode: 0x54, name: "NOP", mode: modeZeroPageX, size: 2, cycles: 4, pageCycles: 0, fn: c.nop},
		{opcode: 0x55, name: "EOR", mode: modeZeroPageX, size: 2, cycles: 4, pageCycles: 0, fn: c.eor},
		{opcode: 0x56, name: "LSR", mode: modeZeroPageX, size: 2, cycles: 6, pageCycles: 0, fn: c.lsr},
		{opcode: 0x57, name: "SRE", mode: modeZeroPageX, size: 2, cycles: 6, pageCycles: 0, fn: c.sre},
		{opcode: 0x58, name: "CLI", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.cli},
		{opcode: 0x59, name: "EOR", mode: modeAbsoluteY, size: 3, cycles: 4, pageCycles: 1, fn: c.eor},
		{opcode: 0x5A, name: "NOP", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.nop},
		{opcode: 0x5B, name: "SRE", mode: modeAbsoluteYW, size: 3, cycles: 7, pageCycles: 0, fn: c.sre},
		{opcode: 0x5C, name: "NOP", mode: modeAbsoluteX, size: 3, cycles: 4, pageCycles: 1, fn: c.nop},
		{opcode: 0x5D, name: "EOR", mode: modeAbsoluteX, size: 3, cycles: 4, pageCycles: 1, fn: c.eor},
		{opcode: 0x5E, name: "LSR", mode: modeAbsoluteXW, size: 3, cycles: 7, pageCycles: 0, fn: c.lsr},
		{opcode: 0x5F, name: "SRE", mode: modeAbsoluteXW, size: 3, cycles: 7, pageCycles: 0, fn: c.sre},
		{opcode: 0x60, name: "RTS", mode: modeImplied, size: 1, cycles: 6, pageCycles: 0, fn: c.rts},
		{opcode: 0x61, name: "ADC", mode: modeIndirectX, size: 2, cycles: 6, pageCycles: 0, fn: c.adc},
		{opcode: 0x62, name: "KIL", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.kil},
		{opcode: 0x63, name: "RRA", mode: modeIndirectX, size: 2, cycles: 8, pageCycles: 0, fn: c.rra},
		{opcode: 0x64, name: "NOP", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.nop},
		{opcode: 0x65, name: "ADC", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.adc},
		{opcode: 0x66, name: "ROR", mode: modeZeroPage, size: 2, cycles: 5, pageCycles: 0, fn: c.ror},
		{opcode: 0x67, name: "RRA", mode: modeZeroPage, size: 2, cycles: 5, pageCycles: 0, fn: c.rra},
		{opcode: 0x68, name: "PLA", mode: modeImplied, size: 1, cycles: 4, pageCycles: 0, fn: c.pla},
		{opcode: 0x69, name: "ADC", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.adc},
		{opcode: 0x6A, name: "ROR", mode: modeAccumulator, size: 1, cycles: 2, pageCycles: 0, fn: c.ror},
		{opcode: 0x6B, name: "ARR", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.arr},
		{opcode: 0x6C, name: "JMP", mode: modeIndirect, size: 3, cycles: 5, pageCycles: 0, fn: c.jmp},
		{opcode: 0x6D, name: "ADC", mode: modeAbsolute, size: 3, cycles: 4, pageCycles: 0, fn: c.adc},
		{opcode: 0x6E, name: "ROR", mode: modeAbsolute, size: 3, cycles: 6, pageCycles: 0, fn: c.ror},
		{opcode: 0x6F, name: "RRA", mode: modeAbsolute, size: 3, cycles: 6, pageCycles: 0, fn: c.rra},
		{opcode: 0x70, name: "BVS", mode: modeRelative, size: 2, cycles: 2, pageCycles: 1, fn: c.bvs},
		{opcode: 0x71, name: "ADC", mode: modeIndirectY, size: 2, cycles: 5, pageCycles: 1, fn: c.adc},
		{opcode: 0x72, name: "KIL", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.kil},
		{opcode: 0x73, name: "RRA", mode: modeIndirectYW, size: 2, cycles: 8, pageCycles: 0, fn: c.rra},
		{opcode: 0x74, name: "NOP", mode: modeZeroPageX, size: 2, cycles: 4, pageCycles: 0, fn: c.nop},
		{opcode: 0x75, name: "ADC", mode: modeZeroPageX, size: 2, cycles: 4, pageCycles: 0, fn: c.adc},
		{opcode: 0x76, name: "ROR", mode: modeZeroPageX, size: 2, cycles: 6, pageCycles: 0, fn: c.ror},
		{opcode: 0x77, name: "RRA", mode: modeZeroPageX, size: 2, cycles: 6, pageCycles: 0, fn: c.rra},
		{opcode: 0x78, name: "SEI", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.sei},
		{opcode: 0x79, name: "ADC", mode: modeAbsoluteY, size: 3, cycles: 4, pageCycles: 1, fn: c.adc},
		{opcode: 0x7A, name: "NOP", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.nop},
		{opcode: 0x7B, name: "RRA", mode: modeAbsoluteYW, size: 3, cycles: 7, pageCycles: 0, fn: c.rra},
		{opcode: 0x7C, name: "NOP", mode: modeAbsoluteX, size: 3, cycles: 4, pageCycles: 1, fn: c.nop},
		{opcode: 0x7D, name: "ADC", mode: modeAbsoluteX, size: 3, cycles: 4, pageCycles: 1, fn: c.adc},
		{opcode: 0x7E, name: "ROR", mode: modeAbsoluteXW, size: 3, cycles: 7, pageCycles: 0, fn: c.ror},
		{opcode: 0x7F, name: "RRA", mode: modeAbsoluteXW, size: 3, cycles: 7, pageCycles: 0, fn: c.rra},
		{opcode: 0x80, name: "NOP", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.nop},
		{opcode: 0x81, name: "STA", mode: modeIndirectX, size: 2, cycles: 6, pageCycles: 0, fn: c.sta},
		{opcode: 0x82, name: "NOP", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.nop},
		{opcode: 0x83, name: "SAX", mode: modeIndirectX, size: 2, cycles: 6, pageCycles: 0, fn: c.sax},
		{opcode: 0x84, name: "STY", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.sty},
		{opcode: 0x85, name: "STA", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.sta},
		{opcode: 0x86, name: "STX", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.stx},
		{opcode: 0x87, name: "SAX", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.sax},
		{opcode: 0x88, name: "DEY", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.dey},
		{opcode: 0x89, name: "NOP", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.nop},
		{opcode: 0x8A, name: "TXA", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.txa},
		{opcode: 0x8B, name: "XAA", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.xaa},
		{opcode: 0x8C, name: "STY", mode: modeAbsolute, size: 3, cycles: 4, pageCycles: 0, fn: c.sty},
		{opcode: 0x8D, name: "STA", mode: modeAbsolute, size: 3, cycles: 4, pageCycles: 0, fn: c.sta},
		{opcode: 0x8E, name: "STX", mode: modeAbsolute, size: 3, cycles: 4, pageCycles: 0, fn: c.stx},
		{opcode: 0x8F, name: "SAX", mode: modeAbsolute, size: 3, cycles: 4, pageCycles: 0, fn: c.sax},
		{opcode: 0x90, name: "BCC", mode: modeRelative, size: 2, cycles: 2, pageCycles: 1, fn: c.bcc},
		{opcode: 0x91, name: "STA", mode: modeIndirectYW, size: 2, cycles: 6, pageCycles: 0, fn: c.sta},
		{opcode: 0x92, name: "KIL", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.kil},
		{opcode: 0x93, name: "AHX", mode: modeIndirectYW, size: 2, cycles: 6, pageCycles: 0, fn: c.ahx},
		{opcode: 0x94, name: "STY", mode: modeZeroPageX, size: 2, cycles: 4, pageCycles: 0, fn: c.sty},
		{opcode: 0x95, name: "STA", mode: modeZeroPageX, size: 2, cycles: 4, pageCycles: 0, fn: c.sta},
		{opcode: 0x96, name: "STX", mode: modeZeroPageY, size: 2, cycles: 4, pageCycles: 0, fn: c.stx},
		{opcode: 0x97, name: "SAX", mode: modeZeroPageY, size: 2, cycles: 4, pageCycles: 0, fn: c.sax},
		{opcode: 0x98, name: "TYA", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.tya},
		{opcode: 0x99, name: "STA", mode: modeAbsoluteYW, size: 3, cycles: 5, pageCycles: 0, fn: c.sta},
		{opcode: 0x9A, name: "TXS", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.txs},
		{opcode: 0x9B, name: "TAS", mode: modeAbsoluteYW, size: 3, cycles: 5, pageCycles: 0, fn: c.tas},
		{opcode: 0x9C, name: "SHY", mode: modeAbsoluteXW, size: 3, cycles: 5, pageCycles: 0, fn: c.shy},
		{opcode: 0x9D, name: "STA", mode: modeAbsoluteXW, size: 3, cycles: 5, pageCycles: 0, fn: c.sta},
		{opcode: 0x9E, name: "SHX", mode: modeAbsoluteYW, size: 3, cycles: 5, pageCycles: 0, fn: c.shx},
		{opcode: 0x9F, name: "AHX", mode: modeAbsoluteYW, size: 3, cycles: 5, pageCycles: 0, fn: c.ahx},
		{opcode: 0xA0, name: "LDY", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.ldy},
		{opcode: 0xA1, name: "LDA", mode: modeIndirectX, size: 2, cycles: 6, pageCycles: 0, fn: c.lda},
		{opcode: 0xA2, name: "LDX", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.ldx},
		{opcode: 0xA3, name: "LAX", mode: modeIndirectX, size: 2, cycles: 6, pageCycles: 0, fn: c.lax},
		{opcode: 0xA4, name: "LDY", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.ldy},
		{opcode: 0xA5, name: "LDA", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.lda},
		{opcode: 0xA6, name: "LDX", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.ldx},
		{opcode: 0xA7, name: "LAX", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.lax},
		{opcode: 0xA8, name: "TAY", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.tay},
		{opcode: 0xA9, name: "LDA", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.lda},
		{opcode: 0xAA, name: "TAX", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.tax},
		{opcode: 0xAB, name: "LAX", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.lax},
		{opcode: 0xAC, name: "LDY", mode: modeAbsolute, size: 3, cycles: 4, pageCycles: 0, fn: c.ldy},
		{opcode: 0xAD, name: "LDA", mode: modeAbsolute, size: 3, cycles: 4, pageCycles: 0, fn: c.lda},
		{opcode: 0xAE, name: "LDX", mode: modeAbsolute, size: 3, cycles: 4, pageCycles: 0, fn: c.ldx},
		{opcode: 0xAF, name: "LAX", mode: modeAbsolute, size: 3, cycles: 4, pageCycles: 0, fn: c.lax},
		{opcode: 0xB0, name: "BCS", mode: modeRelative, size: 2, cycles: 2, pageCycles: 1, fn: c.bcs},
		{opcode: 0xB1, name: "LDA", mode: modeIndirectY, size: 2, cycles: 5, pageCycles: 1, fn: c.lda},
		{opcode: 0xB2, name: "KIL", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.kil},
		{opcode: 0xB3, name: "LAX", mode: modeIndirectY, size: 2, cycles: 5, pageCycles: 1, fn: c.lax},
		{opcode: 0xB4, name: "LDY", mode: modeZeroPageX, size: 2, cycles: 4, pageCycles: 0, fn: c.ldy},
		{opcode: 0xB5, name: "LDA", mode: modeZeroPageX, size: 2, cycles: 4, pageCycles: 0, fn: c.lda},
		{opcode: 0xB6, name: "LDX", mode: modeZeroPageY, size: 2, cycles: 4, pageCycles: 0, fn: c.ldx},
		{opcode: 0xB7, name: "LAX", mode: modeZeroPageY, size: 2, cycles: 4, pageCycles: 0, fn: c.lax},
		{opcode: 0xB8, name: "CLV", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.clv},
		{opcode: 0xB9, name: "LDA", mode: modeAbsoluteY, size: 3, cycles: 4, pageCycles: 1, fn: c.lda},
		{opcode: 0xBA, name: "TSX", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.tsx},
		{opcode: 0xBB, name: "LAS", mode: modeAbsoluteY, size: 3, cycles: 4, pageCycles: 1, fn: c.las},
		{opcode: 0xBC, name: "LDY", mode: modeAbsoluteX, size: 3, cycles: 4, pageCycles: 1, fn: c.ldy},
		{opcode: 0xBD, name: "LDA", mode: modeAbsoluteX, size: 3, cycles: 4, pageCycles: 1, fn: c.lda},
		{opcode: 0xBE, name: "LDX", mode: modeAbsoluteY, size: 3, cycles: 4, pageCycles: 1, fn: c.ldx},
		{opcode: 0xBF, name: "LAX", mode: modeAbsoluteY, size: 3, cycles: 4, pageCycles: 1, fn: c.lax},
		{opcode: 0xC0, name: "CPY", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.cpy},
		{opcode: 0xC1, name: "CPA", mode: modeIndirectX, size: 2, cycles: 6, pageCycles: 0, fn: c.cpa},
		{opcode: 0xC2, name: "NOP", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.nop},
		{opcode: 0xC3, name: "DCP", mode: modeIndirectX, size: 2, cycles: 8, pageCycles: 0, fn: c.dcp},
		{opcode: 0xC4, name: "CPY", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.cpy},
		{opcode: 0xC5, name: "CPA", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.cpa},
		{opcode: 0xC6, name: "DEC", mode: modeZeroPage, size: 2, cycles: 5, pageCycles: 0, fn: c.dec},
		{opcode: 0xC7, name: "DCP", mode: modeZeroPage, size: 2, cycles: 5, pageCycles: 0, fn: c.dcp},
		{opcode: 0xC8, name: "INY", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.iny},
		{opcode: 0xC9, name: "CPA", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.cpa},
		{opcode: 0xCA, name: "DEX", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.dex},
		{opcode: 0xCB, name: "AXS", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.axs},
		{opcode: 0xCC, name: "CPY", mode: modeAbsolute, size: 3, cycles: 4, pageCycles: 0, fn: c.cpy},
		{opcode: 0xCD, name: "CPA", mode: modeAbsolute, size: 3, cycles: 4, pageCycles: 0, fn: c.cpa},
		{opcode: 0xCE, name: "DEC", mode: modeAbsolute, size: 3, cycles: 6, pageCycles: 0, fn: c.dec},
		{opcode: 0xCF, name: "DCP", mode: modeAbsolute, size: 3, cycles: 6, pageCycles: 0, fn: c.dcp},
		{opcode: 0xD0, name: "BNE", mode: modeRelative, size: 2, cycles: 2, pageCycles: 1, fn: c.bne},
		{opcode: 0xD1, name: "CPA", mode: modeIndirectY, size: 2, cycles: 5, pageCycles: 1, fn: c.cpa},
		{opcode: 0xD2, name: "KIL", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.kil},
		{opcode: 0xD3, name: "DCP", mode: modeIndirectYW, size: 2, cycles: 8, pageCycles: 0, fn: c.dcp},
		{opcode: 0xD4, name: "NOP", mode: modeZeroPageX, size: 2, cycles: 4, pageCycles: 0, fn: c.nop},
		{opcode: 0xD5, name: "CPA", mode: modeZeroPageX, size: 2, cycles: 4, pageCycles: 0, fn: c.cpa},
		{opcode: 0xD6, name: "DEC", mode: modeZeroPageX, size: 2, cycles: 6, pageCycles: 0, fn: c.dec},
		{opcode: 0xD7, name: "DCP", mode: modeZeroPageX, size: 2, cycles: 6, pageCycles: 0, fn: c.dcp},
		{opcode: 0xD8, name: "CLD", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.cld},
		{opcode: 0xD9, name: "CPA", mode: modeAbsoluteY, size: 3, cycles: 4, pageCycles: 1, fn: c.cpa},
		{opcode: 0xDA, name: "NOP", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.nop},
		{opcode: 0xDB, name: "DCP", mode: modeAbsoluteYW, size: 3, cycles: 7, pageCycles: 0, fn: c.dcp},
		{opcode: 0xDC, name: "NOP", mode: modeAbsoluteX, size: 3, cycles: 4, pageCycles: 1, fn: c.nop},
		{opcode: 0xDD, name: "CPA", mode: modeAbsoluteX, size: 3, cycles: 4, pageCycles: 1, fn: c.cpa},
		{opcode: 0xDE, name: "DEC", mode: modeAbsoluteXW, size: 3, cycles: 7, pageCycles: 0, fn: c.dec},
		{opcode: 0xDF, name: "DCP", mode: modeAbsoluteXW, size: 3, cycles: 7, pageCycles: 0, fn: c.dcp},
		{opcode: 0xE0, name: "CPX", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.cpx},
		{opcode: 0xE1, name: "SBC", mode: modeIndirectX, size: 2, cycles: 6, pageCycles: 0, fn: c.sbc},
		{opcode: 0xE2, name: "NOP", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.nop},
		{opcode: 0xE3, name: "ISC", mode: modeIndirectX, size: 2, cycles: 8, pageCycles: 0, fn: c.isc},
		{opcode: 0xE4, name: "CPX", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.cpx},
		{opcode: 0xE5, name: "SBC", mode: modeZeroPage, size: 2, cycles: 3, pageCycles: 0, fn: c.sbc},
		{opcode: 0xE6, name: "INC", mode: modeZeroPage, size: 2, cycles: 5, pageCycles: 0, fn: c.inc},
		{opcode: 0xE7, name: "ISC", mode: modeZeroPage, size: 2, cycles: 5, pageCycles: 0, fn: c.isc},
		{opcode: 0xE8, name: "INX", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.inx},
		{opcode: 0xE9, name: "SBC", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.sbc},
		{opcode: 0xEA, name: "NOP", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.nop},
		{opcode: 0xEB, name: "SBC", mode: modeImmediate, size: 2, cycles: 2, pageCycles: 0, fn: c.sbc},
		{opcode: 0xEC, name: "CPX", mode: modeAbsolute, size: 3, cycles: 4, pageCycles: 0, fn: c.cpx},
		{opcode: 0xED, name: "SBC", mode: modeAbsolute, size: 3, cycles: 4, pageCycles: 0, fn: c.sbc},
		{opcode: 0xEE, name: "INC", mode: modeAbsolute, size: 3, cycles: 6, pageCycles: 0, fn: c.inc},
		{opcode: 0xEF, name: "ISC", mode: modeAbsolute, size: 3, cycles: 6, pageCycles: 0, fn: c.isc},
		{opcode: 0xF0, name: "BEQ", mode: modeRelative, size: 2, cycles: 2, pageCycles: 1, fn: c.beq},
		{opcode: 0xF1, name: "SBC", mode: modeIndirectY, size: 2, cycles: 5, pageCycles: 1, fn: c.sbc},
		{opcode: 0xF2, name: "KIL", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.kil},
		{opcode: 0xF3, name: "ISC", mode: modeIndirectYW, size: 2, cycles: 8, pageCycles: 0, fn: c.isc},
		{opcode: 0xF4, name: "NOP", mode: modeZeroPageX, size: 2, cycles: 4, pageCycles: 0, fn: c.nop},
		{opcode: 0xF5, name: "SBC", mode: modeZeroPageX, size: 2, cycles: 4, pageCycles: 0, fn: c.sbc},
		{opcode: 0xF6, name: "INC", mode: modeZeroPageX, size: 2, cycles: 6, pageCycles: 0, fn: c.inc},
		{opcode: 0xF7, name: "ISC", mode: modeZeroPageX, size: 2, cycles: 6, pageCycles: 0, fn: c.isc},
		{opcode: 0xF8, name: "SED", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.sed},
		{opcode: 0xF9, name: "SBC", mode: modeAbsoluteY, size: 3, cycles: 4, pageCycles: 1, fn: c.sbc},
		{opcode: 0xFA, name: "NOP", mode: modeImplied, size: 1, cycles: 2, pageCycles: 0, fn: c.nop},
		{opcode: 0xFB, name: "ISC", mode: modeAbsoluteYW, size: 3, cycles: 7, pageCycles: 0, fn: c.isc},
		{opcode: 0xFC, name: "NOP", mode: modeAbsoluteX, size: 3, cycles: 4, pageCycles: 1, fn: c.nop},
		{opcode: 0xFD, name: "SBC", mode: modeAbsoluteX, size: 3, cycles: 4, pageCycles: 1, fn: c.sbc},
		{opcode: 0xFE, name: "INC", mode: modeAbsoluteXW, size: 3, cycles: 7, pageCycles: 0, fn: c.inc},
		{opcode: 0xFF, name: "ISC", mode: modeAbsoluteXW, size: 3, cycles: 7, pageCycles: 0, fn: c.isc},
	}
}

// ADC - Add with Carry
func (cpu *CPU) adc(pc uint16, mode AddressingMode) {
	cpu.ADD(cpu.GetOperandValue(mode))
}

// AND - Logical AND
func (cpu *CPU) and(pc uint16, mode AddressingMode) {
	cpu.SetA(cpu.state.A & cpu.GetOperandValue(mode))
}

// ASL - Arithmetic Shift Left
func (cpu *CPU) asl(pc uint16, mode AddressingMode) {
	switch mode {
	case modeAccumulator:
		cpu.SetA(cpu.ASL(cpu.state.A))
	default:
		addr := cpu.currentOperand
		value := cpu.ReadMemory(addr, MemoryRead)
		cpu.WriteMemory(addr, value, DummyWrite)
		cpu.WriteMemory(addr, cpu.ASL(value), MemoryWrite)
	}
}

// BCC - Branch if Carry Clear
func (cpu *CPU) bcc(pc uint16, mode AddressingMode) {
	cpu.BranchRelative(!cpu.CheckFlag(PSFlagsCarry))
}

// BCS - Branch if Carry Set
func (cpu *CPU) bcs(pc uint16, mode AddressingMode) {
	cpu.BranchRelative(cpu.CheckFlag(PSFlagsCarry))
}

// BEQ - Branch if Equal
func (cpu *CPU) beq(pc uint16, mode AddressingMode) {
	cpu.BranchRelative(cpu.CheckFlag(PSFlagsZero))
}

// BIT - Bit Test
func (cpu *CPU) bit(pc uint16, mode AddressingMode) {
	value := cpu.GetOperandValue(mode)
	cpu.ClearFlags(PSFlagsZero | PSFlagsOverflow | PSFlagsNegative)
	if (cpu.state.A & value) == 0 {
		cpu.SetFlags(PSFlagsZero)
	}
	if (value & 0x40) > 0 {
		cpu.SetFlags(PSFlagsOverflow)
	}
	if (value & 0x80) > 0 {
		cpu.SetFlags(PSFlagsNegative)
	}
}

// BMI - Branch if Minus
func (cpu *CPU) bmi(pc uint16, mode AddressingMode) {
	cpu.BranchRelative(cpu.CheckFlag(PSFlagsNegative))
}

// BNE - Branch if Not Equal
func (cpu *CPU) bne(pc uint16, mode AddressingMode) {
	cpu.BranchRelative(!cpu.CheckFlag(PSFlagsZero))
}

// BPL - Branch if Positive
func (cpu *CPU) bpl(pc uint16, mode AddressingMode) {
	cpu.BranchRelative(!cpu.CheckFlag(PSFlagsNegative))
}

// BRK - Force Interrupt
func (cpu *CPU) brk(pc uint16, mode AddressingMode) {
	cpu.push16(cpu.state.PC + 1)

	flags := cpu.Flags() | PSFlagsBreak | PSFlagsReserved
	if cpu.needNMI {
		cpu.needNMI = false
		cpu.push(flags)
		cpu.SetFlags(PSFlagsInterrupt)

		cpu.SetPC(cpu.ReadMemory16(NMIVector, MemoryRead))
	} else {
		cpu.push(flags)
		cpu.SetFlags(PSFlagsInterrupt)

		cpu.SetPC(cpu.ReadMemory16(IRQVector, MemoryRead))
	}

	cpu.prevNeedNMI = false
}

// BVC - Branch if Overflow Clear
func (cpu *CPU) bvc(pc uint16, mode AddressingMode) {
	cpu.BranchRelative(!cpu.CheckFlag(PSFlagsOverflow))
}

// BVS - Branch if Overflow Set
func (cpu *CPU) bvs(pc uint16, mode AddressingMode) {
	cpu.BranchRelative(cpu.CheckFlag(PSFlagsOverflow))
}

// CLC - Clear Carry Flag
func (cpu *CPU) clc(pc uint16, mode AddressingMode) {
	cpu.state.C = 0
}

// CLD - Clear Decimal Mode
func (cpu *CPU) cld(pc uint16, mode AddressingMode) {
	cpu.state.D = 0
}

// CLI - Clear Interrupt Disable
func (cpu *CPU) cli(pc uint16, mode AddressingMode) {
	cpu.state.I = 0
}

// CLV - Clear Overflow Flag
func (cpu *CPU) clv(pc uint16, mode AddressingMode) {
	cpu.state.V = 0
}

// CMA(CMP) - Compare
func (cpu *CPU) cpa(pc uint16, mode AddressingMode) {
	cpu.CMP(cpu.state.A, cpu.GetOperandValue(mode))
}

// CPX - Compare X Register
func (cpu *CPU) cpx(pc uint16, mode AddressingMode) {
	cpu.CMP(cpu.state.X, cpu.GetOperandValue(mode))
}

// CPY - Compare Y Register
func (cpu *CPU) cpy(pc uint16, mode AddressingMode) {
	cpu.CMP(cpu.state.Y, cpu.GetOperandValue(mode))
}

// DEC - Decrement Memory
func (cpu *CPU) dec(pc uint16, mode AddressingMode) {
	addr := cpu.currentOperand
	cpu.ClearFlags(PSFlagsNegative | PSFlagsZero)
	value := cpu.ReadMemory(addr, MemoryRead)
	cpu.WriteMemory(addr, value, DummyWrite)

	value--
	cpu.setZN(value)
	cpu.WriteMemory(addr, value, DummyWrite)
}

// DEX - Decrement X Register
func (cpu *CPU) dex(pc uint16, mode AddressingMode) {
	cpu.SetX(cpu.state.X - 1)
}

// DEY - Decrement Y Register
func (cpu *CPU) dey(pc uint16, mode AddressingMode) {
	cpu.SetY(cpu.state.Y - 1)
}

// EOR - Exclusive OR
func (cpu *CPU) eor(pc uint16, mode AddressingMode) {
	cpu.SetA(cpu.state.A ^ cpu.GetOperandValue(mode))
}

// INC - Increment Memory
func (cpu *CPU) inc(pc uint16, mode AddressingMode) {
	addr := cpu.currentOperand
	cpu.ClearFlags(PSFlagsNegative | PSFlagsZero)
	value := cpu.ReadMemory(addr, MemoryRead)

	cpu.WriteMemory(addr, value, DummyWrite)

	value++
	cpu.setZN(value)
	cpu.WriteMemory(addr, value, MemoryWrite)
}

// INX - Increment X Register
func (cpu *CPU) inx(pc uint16, mode AddressingMode) {
	cpu.SetX(cpu.state.X + 1)
}

// INY - Increment Y Register
func (cpu *CPU) iny(pc uint16, mode AddressingMode) {
	cpu.SetY(cpu.state.Y + 1)
}

// JMP - Jump
func (cpu *CPU) jmp(pc uint16, mode AddressingMode) {
	switch mode {
	case modeAbsolute:
		cpu.SetPC(cpu.currentOperand)
	case modeIndirect:
		operand := cpu.currentOperand
		var addr uint16
		if (operand & 0xFF) == 0xFF {
			lo := cpu.ReadMemory(operand, MemoryRead)
			hi := cpu.ReadMemory(operand-0xFF, MemoryRead)
			addr = (uint16(hi) << 8) | uint16(lo)
		} else {
			addr = cpu.ReadMemory16(operand, MemoryRead)
		}
		cpu.SetPC(addr)
	default:
		log.Fatalln("Illegal Instruction JMP")
	}
}

// JSR - Jump to Subroutine
func (cpu *CPU) jsr(pc uint16, mode AddressingMode) {
	addr := cpu.currentOperand
	cpu.ReadDummy()
	cpu.push16(cpu.state.PC - 1)
	cpu.SetPC(addr)
}

// LDA - Load Accumulator
func (cpu *CPU) lda(pc uint16, mode AddressingMode) {
	cpu.SetA(cpu.GetOperandValue(mode))
}

// LDX - Load X Register
func (cpu *CPU) ldx(pc uint16, mode AddressingMode) {
	cpu.SetX(cpu.GetOperandValue(mode))
}

// LDY - Load Y Register
func (cpu *CPU) ldy(pc uint16, mode AddressingMode) {
	cpu.SetY(cpu.GetOperandValue(mode))
}

// LSR - Logical Shift Right
func (cpu *CPU) lsr(pc uint16, mode AddressingMode) {
	switch mode {
	case modeAccumulator:
		cpu.SetA(cpu.LSR(cpu.state.A))
	default:
		addr := cpu.currentOperand
		value := cpu.ReadMemory(addr, MemoryRead)
		cpu.WriteMemory(addr, value, DummyWrite)
		cpu.WriteMemory(addr, cpu.LSR(value), DummyWrite)
	}
}

// NOP - No Operation
func (cpu *CPU) nop(pc uint16, mode AddressingMode) {
	//Make sure the nop operation takes as many cycles as meant to
	cpu.GetOperandValue(mode)
}

// ORA - Logical Inclusive OR
func (cpu *CPU) ora(pc uint16, mode AddressingMode) {
	cpu.SetA(cpu.state.A | cpu.GetOperandValue(mode))
}

// PHA - Push Accumulator
func (cpu *CPU) pha(pc uint16, mode AddressingMode) {
	cpu.push(cpu.state.A)
}

// PHP - Push Processor Status
func (cpu *CPU) php(pc uint16, mode AddressingMode) {
	flags := cpu.Flags() | PSFlagsBreak | PSFlagsReserved
	cpu.push(flags)
}

// PLA - Pull Accumulator
func (cpu *CPU) pla(pc uint16, mode AddressingMode) {
	cpu.ReadDummy()
	cpu.SetA(cpu.pull())
}

// PLP - Pull Processor Status
func (cpu *CPU) plp(pc uint16, mode AddressingMode) {
	cpu.ReadDummy()
	cpu.SetPS(cpu.pull())
}

// ROL - Rotate Left
func (cpu *CPU) rol(pc uint16, mode AddressingMode) {
	switch mode {
	case modeAccumulator:
		cpu.SetA(cpu.ROL(cpu.state.A))
	default:
		addr := cpu.currentOperand
		value := cpu.ReadMemory(addr, MemoryRead)
		cpu.WriteMemory(addr, value, DummyWrite)
		cpu.WriteMemory(addr, cpu.ROL(value), MemoryWrite)
	}
}

// ROR - Rotate Right
func (cpu *CPU) ror(pc uint16, mode AddressingMode) {
	switch mode {
	case modeAccumulator:
		cpu.SetA(cpu.ROR(cpu.state.A))
	default:
		addr := cpu.currentOperand
		value := cpu.ReadMemory(addr, MemoryRead)
		cpu.WriteMemory(addr, value, DummyWrite)
		cpu.WriteMemory(addr, cpu.ROR(value), MemoryWrite)
	}
}

// RTI - Return from Interrupt
func (cpu *CPU) rti(pc uint16, mode AddressingMode) {
	cpu.ReadDummy()
	cpu.SetPS(cpu.pull())
	cpu.SetPC(cpu.pull16())
}

// RTS - Return from Subroutine
func (cpu *CPU) rts(pc uint16, mode AddressingMode) {
	addr := cpu.pull16()
	cpu.ReadDummy()
	cpu.ReadDummy()
	cpu.SetPC(addr + 1)
}

// SBC - Subtract with Carry
func (cpu *CPU) sbc(pc uint16, mode AddressingMode) {
	cpu.ADD(cpu.GetOperandValue(mode) ^ 0xFF)
}

// SEC - Set Carry Flag
func (cpu *CPU) sec(pc uint16, mode AddressingMode) {
	cpu.state.C = 1
}

// SED - Set Decimal Flag
func (cpu *CPU) sed(pc uint16, mode AddressingMode) {
	cpu.state.D = 1
}

// SEI - Set Interrupt Disable
func (cpu *CPU) sei(pc uint16, mode AddressingMode) {
	cpu.state.I = 1
}

// STA - Store Accumulator
func (cpu *CPU) sta(pc uint16, mode AddressingMode) {
	cpu.WriteMemory(cpu.currentOperand, cpu.state.A, MemoryWrite)
}

// STX - Store X Register
func (cpu *CPU) stx(pc uint16, mode AddressingMode) {
	cpu.WriteMemory(cpu.currentOperand, cpu.state.X, MemoryWrite)
}

// STY - Store Y Register
func (cpu *CPU) sty(pc uint16, mode AddressingMode) {
	cpu.WriteMemory(cpu.currentOperand, cpu.state.Y, MemoryWrite)
}

// TAX - Transfer Accumulator to X
func (cpu *CPU) tax(pc uint16, mode AddressingMode) {
	cpu.SetX(cpu.state.A)
}

// TAY - Transfer Accumulator to Y
func (cpu *CPU) tay(pc uint16, mode AddressingMode) {
	cpu.SetY(cpu.state.A)
}

// TSX - Transfer Stack Pointer to X
func (cpu *CPU) tsx(pc uint16, mode AddressingMode) {
	cpu.SetX(cpu.state.SP)
}

// TXA - Transfer X to Accumulator
func (cpu *CPU) txa(pc uint16, mode AddressingMode) {
	cpu.SetA(cpu.state.X)
}

// TXS - Transfer X to Stack Pointer
func (cpu *CPU) txs(pc uint16, mode AddressingMode) {
	cpu.SetSP(cpu.state.X)
}

// TYA - Transfer Y to Accumulator
func (cpu *CPU) tya(pc uint16, mode AddressingMode) {
	cpu.SetA(cpu.state.Y)
}

// illegal opcodes below

func (cpu *CPU) ahx(pc uint16, mode AddressingMode) {
	log.Fatalln("Implemented yet (ahx)")
}

func (cpu *CPU) alr(pc uint16, mode AddressingMode) {
	log.Fatalln("Implemented yet (alr)")
}

func (cpu *CPU) anc(pc uint16, mode AddressingMode) {
	log.Fatalln("Implemented yet (anc)")
}

func (cpu *CPU) arr(pc uint16, mode AddressingMode) {
	log.Fatalln("Implemented yet (arr)")
}

func (cpu *CPU) axs(pc uint16, mode AddressingMode) {
	log.Fatalln("Implemented yet (axs)")
}

func (cpu *CPU) dcp(pc uint16, mode AddressingMode) {
	value := cpu.GetOperandValue(mode)
	cpu.WriteMemory(cpu.currentOperand, value, DummyWrite)
	value--
	cpu.CMP(cpu.state.A, value)
	cpu.WriteMemory(cpu.currentOperand, value, MemoryWrite)
}

// ISB, ISC
func (cpu *CPU) isc(pc uint16, mode AddressingMode) {
	value := cpu.GetOperandValue(mode)
	cpu.WriteMemory(cpu.currentOperand, value, DummyWrite)
	value++
	cpu.ADD(value ^ 0xFF)
	cpu.WriteMemory(cpu.currentOperand, value, MemoryWrite)
}

func (cpu *CPU) kil(pc uint16, mode AddressingMode) {
	log.Fatalln("Implemented yet (kil)")
}

func (cpu *CPU) las(pc uint16, mode AddressingMode) {
	log.Fatalln("Implemented yet (las)")
}

func (cpu *CPU) lax(pc uint16, mode AddressingMode) {
	value := cpu.GetOperandValue(mode)
	cpu.SetX(value)
	cpu.SetA(value)
}

func (cpu *CPU) rla(pc uint16, mode AddressingMode) {
	value := cpu.GetOperandValue(mode)
	cpu.WriteMemory(cpu.currentOperand, value, DummyWrite)
	shiftedValue := cpu.ROL(value)
	cpu.SetA(cpu.state.A & shiftedValue)
	cpu.WriteMemory(cpu.currentOperand, shiftedValue, MemoryWrite)
}

func (cpu *CPU) rra(pc uint16, mode AddressingMode) {
	value := cpu.GetOperandValue(mode)
	cpu.WriteMemory(cpu.currentOperand, value, DummyWrite)
	shiftedValue := cpu.ROR(value)
	cpu.ADD(shiftedValue)
	cpu.WriteMemory(cpu.currentOperand, shiftedValue, MemoryWrite)
}

func (cpu *CPU) sax(pc uint16, mode AddressingMode) {
	cpu.WriteMemory(cpu.currentOperand, (cpu.state.A & cpu.state.X), MemoryWrite)
}

func (cpu *CPU) shx(pc uint16, mode AddressingMode) {
	log.Fatalln("Implemented yet (shx)")
}

func (cpu *CPU) shy(pc uint16, mode AddressingMode) {
	log.Fatalln("Implemented yet (shy)")
}

func (cpu *CPU) slo(pc uint16, mode AddressingMode) {
	value := cpu.GetOperandValue(mode)
	cpu.WriteMemory(cpu.currentOperand, value, DummyWrite)
	shiftedValue := cpu.ASL(value)
	cpu.SetA(cpu.state.A | shiftedValue)
	cpu.WriteMemory(cpu.currentOperand, shiftedValue, MemoryWrite)
}

func (cpu *CPU) sre(pc uint16, mode AddressingMode) {
	value := cpu.GetOperandValue(mode)
	cpu.WriteMemory(cpu.currentOperand, value, DummyWrite)
	shiftedValue := cpu.LSR(value)
	cpu.SetA(cpu.state.A ^ shiftedValue)
	cpu.WriteMemory(cpu.currentOperand, shiftedValue, MemoryWrite)
}

func (cpu *CPU) tas(pc uint16, mode AddressingMode) {
	log.Fatalln("Implemented yet (tas)")
}

func (cpu *CPU) xaa(pc uint16, mode AddressingMode) {
	log.Fatalln("Implemented yet (xaa)")
}

// Utilities

func (cpu *CPU) GetOperandValue(mode AddressingMode) byte {
	switch mode {
	case modeZeroPage, modeZeroPageX, modeZeroPageY,
		modeIndirect, modeIndirectX, modeIndirectY, modeIndirectYW,
		modeAbsolute, modeAbsoluteX, modeAbsoluteXW, modeAbsoluteY, modeAbsoluteYW:
		return cpu.ReadMemory(cpu.currentOperand, MemoryRead)
	case modeAccumulator, modeImplied, modeImmediate, modeRelative:
		return byte(cpu.currentOperand)
	default:
		log.Fatalln("Unknown addressing mode")
	}

	// (NOT REACH) NOTHING DONE
	return 0x00
}

func (cpu *CPU) SetA(value byte) {
	cpu.ClearFlags(PSFlagsZero | PSFlagsNegative)
	cpu.setZN(value)

	cpu.state.A = value
}

func (cpu *CPU) SetX(value byte) {
	cpu.ClearFlags(PSFlagsZero | PSFlagsNegative)
	cpu.setZN(value)

	cpu.state.X = value
}

func (cpu *CPU) SetY(value byte) {
	cpu.ClearFlags(PSFlagsZero | PSFlagsNegative)
	cpu.setZN(value)

	cpu.state.Y = value
}

func (cpu *CPU) SetSP(value byte) {
	cpu.state.SP = value
}

func (cpu *CPU) SetPS(value byte) {
	cpu.SetAllFlags(value & 0xCF)
}

func (cpu *CPU) SetPC(value uint16) {
	cpu.state.PC = value
}

func (cpu *CPU) ADD(value byte) {
	var c uint16
	if cpu.CheckFlag(PSFlagsCarry) {
		c = PSFlagsCarry
	} else {
		c = 0
	}

	result := uint16(cpu.state.A) + uint16(value) + c
	cpu.ClearFlags(PSFlagsCarry | PSFlagsNegative | PSFlagsOverflow | PSFlagsZero)
	cpu.setZN(byte(result))
	// XXX: Need Refactor
	if (^(cpu.state.A ^ value) & (cpu.state.A ^ byte(result)) & 0x80) > 0 {
		cpu.SetFlags(PSFlagsOverflow)
	}
	if result > 0xFF {
		cpu.SetFlags(PSFlagsCarry)
	}

	cpu.state.A = byte(result)
}

func (cpu *CPU) ASL(value byte) byte {
	cpu.ClearFlags(PSFlagsCarry | PSFlagsNegative | PSFlagsZero)
	if (value & 0x80) > 0 {
		cpu.SetFlags(PSFlagsCarry)
	}

	result := value << 1
	cpu.setZN(result)
	return result
}

func (cpu *CPU) CMP(reg, value byte) {
	cpu.ClearFlags(PSFlagsCarry | PSFlagsNegative | PSFlagsZero)

	result := reg - value

	if reg >= value {
		cpu.SetFlags(PSFlagsCarry)
	}
	if reg == value {
		cpu.SetFlags(PSFlagsZero)
	}
	if (result & 0x80) == 0x80 {
		cpu.SetFlags(PSFlagsNegative)
	}
}

func (cpu *CPU) LSR(value byte) byte {
	cpu.ClearFlags(PSFlagsCarry | PSFlagsNegative | PSFlagsZero)
	if (value & 0x01) > 0 {
		cpu.SetFlags(PSFlagsCarry)
	}

	result := value >> 1
	cpu.setZN(result)
	return result
}

func (cpu *CPU) ROL(value byte) byte {
	carryFlag := cpu.CheckFlag(PSFlagsCarry)
	cpu.ClearFlags(PSFlagsCarry | PSFlagsNegative | PSFlagsZero)

	if (value & 0x80) > 0 {
		cpu.SetFlags(PSFlagsCarry)
	}

	var result byte
	if carryFlag {
		result = (value<<1 | 0x01)
	} else {
		result = (value<<1 | 0x00)
	}
	cpu.setZN(result)
	return result
}

func (cpu *CPU) ROR(value byte) byte {
	carryFlag := cpu.CheckFlag(PSFlagsCarry)
	cpu.ClearFlags(PSFlagsCarry | PSFlagsNegative | PSFlagsZero)

	if (value & 0x01) > 0 {
		cpu.SetFlags(PSFlagsCarry)
	}

	var result byte
	if carryFlag {
		result = (value>>1 | 0x80)
	} else {
		result = (value>>1 | 0x00)
	}
	cpu.setZN(result)
	return result
}

// refs: github.com/fogleman/nes
package toynes

import (
	"image"
)

type Console struct {
	CPU         *CPU
	APU         *APU
	PPU         *PPU
	Cartridge   *Cartridge
	Controller1 *Controller
	Controller2 *Controller

	disableOCnextFrame bool
}

func NewConsole(path string, isNSF bool) (*Console, error) {
	controller1 := NewController()
	controller2 := NewController()
	console := Console{
		CPU:         nil,
		APU:         nil,
		PPU:         nil,
		Cartridge:   nil,
		Controller1: controller1,
		Controller2: controller2,
	}
	console.CPU = NewCPU(&console)
	console.APU = NewAPU(&console)
	console.PPU = NewPPU(&console)

	var cartridge *Cartridge
	var err error
	if isNSF {
		cartridge, err = LoadNSFFile(path, &console)
		if err != nil {
			return nil, err
		}
	} else {
		cartridge, err = LoadNESFile(path, &console)
		if err != nil {
			return nil, err
		}
	}
	console.Cartridge = cartridge

	bus := NewBus(
		console.CPU,
		console.PPU,
		console.APU,
		console.Controller1,
		console.Controller2,
		console.Cartridge,
	)
	console.CPU.bus = bus

	console.Reset()

	return &console, nil
}

func (console *Console) Reset() {
	console.CPU.Reset()
	console.PPU.Reset()
	console.APU.Reset()
}

func (console *Console) Step() int {
	cpuCycles := console.CPU.Step()
	// ppuCycles := cpuCycles * 3
	// for i := 0; i < ppuCycles; i++ {
	// 	console.PPU.Step()
	// }
	// for i := 0; i < cpuCycles; i++ {
	// 	console.APU.Step()
	// 	console.Cartridge.Mapper.Step()
	// }
	return cpuCycles
}

func (console *Console) StepFrame() int {
	cpuCycles := 0
	frame := console.PPU.Frame
	for frame == console.PPU.Frame {
		cpuCycles += console.Step()
	}
	return cpuCycles
}

func (console *Console) StepSeconds(seconds float64) {
	cycles := int(CPUFrequency * seconds)
	for cycles > 0 {
		cycles -= console.Step()
	}
}

func (console *Console) Buffer() *image.RGBA {
	return console.PPU.front
}

func (console *Console) SetButtons1(buttons [8]bool) {
	console.Controller1.SetButtons(buttons)
}

func (console *Console) SetButtons2(buttons [8]bool) {
	console.Controller2.SetButtons(buttons)
}

func (console *Console) SetAudioChannel(channel chan float32) {
	console.APU.channel = channel
}

func (console *Console) SetAudioSampleRate(sampleRate float64) {
	if sampleRate != 0 {
		// Convert samples per second to cpu steps per sample
		console.APU.sampleRate = CPUFrequency / sampleRate
		// Initialize filters
		console.APU.filterChain = APUFilterChain{
			HighPassFilter(float32(sampleRate), 90),
			HighPassFilter(float32(sampleRate), 440),
			LowPassFilter(float32(sampleRate), 14000),
		}
	} else {
		console.APU.filterChain = nil
	}
}

// XXX: really need?
func (console *Console) SetNextFrameOverclockStatus(disabled bool) {
	console.disableOCnextFrame = disabled
}

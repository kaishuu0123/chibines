// refs: github.com/libretro/Mesen
package toynes

const frameCounterRate = CPUFrequency / 240.0

var squareTable [31]float32
var tndTable [203]float32

func init() {
	for i := 0; i < 31; i++ {
		squareTable[i] = 95.52 / (8128.0/float32(i) + 100)
	}
	for i := 0; i < 203; i++ {
		tndTable[i] = 163.67 / (24329.0/float32(i) + 100)
	}
}

type BaseAPUChannel struct {
	previousCycle uint32
	lastOutput    int8
	timer         uint16
	period        uint16
}

func (b *BaseAPUChannel) Reset() {
	b.timer = 0
	b.period = 0
	b.lastOutput = 0
	b.previousCycle = 0
}

func (b *BaseAPUChannel) EndFrame() {
	b.previousCycle = 0
}

type APULengthCounter struct {
	baseAPUChannel             *BaseAPUChannel
	console                    *Console
	lcLookupTable              [32]byte
	newHaltValue               bool
	enabled                    bool
	lengthCounterHalt          bool
	lengthCounter              byte
	lengthCounterReloadValue   byte
	lengthCounterPreviousValue byte
}

func NewAPULengthCounter(console *Console) *APULengthCounter {
	var lengthTable [32]byte = [32]byte{
		10, 254, 20, 2, 40, 4, 80, 6, 160, 8, 60, 10, 14, 12, 26, 14, 12, 16, 24, 18, 48, 20, 96, 22, 192, 24, 72, 26, 16, 28, 32, 30,
	}

	lc := &APULengthCounter{
		baseAPUChannel: &BaseAPUChannel{
			previousCycle: 0,
			lastOutput:    0,
			timer:         0,
			period:        0,
		},
		console:                    console,
		newHaltValue:               false,
		lengthCounterHalt:          false,
		enabled:                    false,
		lengthCounter:              0,
		lengthCounterReloadValue:   0,
		lengthCounterPreviousValue: 0,
	}

	for i := 0; i < len(lc.lcLookupTable); i++ {
		lc.lcLookupTable[i] = lengthTable[i]
	}

	return lc
}

func (lc *APULengthCounter) InitializeLengthCounter(haltFlag bool) {
	lc.console.APU.SetNeedToRun()
	lc.newHaltValue = haltFlag
}

func (lc *APULengthCounter) LoadLengthCounter(value byte) {
	if lc.enabled {
		lc.lengthCounterReloadValue = lc.lcLookupTable[value]
		lc.lengthCounterPreviousValue = lc.lengthCounter
		lc.console.APU.SetNeedToRun()
	}
}

func (lc *APULengthCounter) Reset() {
	lc.enabled = false
	lc.lengthCounterHalt = false
	lc.lengthCounter = 0
	lc.newHaltValue = false
	lc.lengthCounterReloadValue = 0
	lc.lengthCounterPreviousValue = 0
}

func (lc *APULengthCounter) GetStatus() bool {
	return lc.lengthCounter > 0
}

func (lc *APULengthCounter) ReloadCounter() {
	if lc.lengthCounterReloadValue > 0 {
		if lc.lengthCounter == lc.lengthCounterPreviousValue {
			lc.lengthCounter = lc.lengthCounterReloadValue
		}
		lc.lengthCounterReloadValue = 0
	}

	lc.lengthCounterHalt = lc.newHaltValue
}

func (lc *APULengthCounter) TickLengthCounter() {
	if lc.lengthCounter > 0 && lc.lengthCounterHalt == false {
		lc.lengthCounter--
	}
}

func (lc *APULengthCounter) SetEnabled(enabled bool) {
	if enabled == false {
		lc.lengthCounter = 0
	}
	lc.enabled = enabled
}

type APUEnvelope struct {
	apuLengthCounter *APULengthCounter

	constantVolume  bool
	volume          byte
	envelopeCounter byte
	start           bool
	divider         int8
	counter         uint8
}

func NewAPUEnvelope(console *Console) *APUEnvelope {
	return &APUEnvelope{
		apuLengthCounter: NewAPULengthCounter(console),
		constantVolume:   false,
		volume:           0,
		envelopeCounter:  0,
		start:            false,
		divider:          0,
		counter:          0,
	}
}

func (e *APUEnvelope) InitializeEnvelope(regValue byte) {
	e.constantVolume = (regValue & 0x10) == 0x10
	e.volume = regValue & 0x0F
}

func (e *APUEnvelope) ResetEnvelope() {
	e.start = true
}

func (e *APUEnvelope) GetVolume() uint32 {
	if e.apuLengthCounter.lengthCounter > 0 {
		if e.constantVolume {
			return uint32(e.volume)
		} else {
			return uint32(e.counter)
		}
	} else {
		return 0
	}
}

func (e *APUEnvelope) Reset() {
	e.apuLengthCounter.Reset()

	e.constantVolume = false
	e.volume = 0
	e.envelopeCounter = 0
	e.start = false
	e.divider = 0
	e.counter = 0
}

func (e *APUEnvelope) TickEnvelope() {
	if e.start == false {
		e.divider--
		if e.divider < 0 {
			e.divider = int8(e.volume)
			if e.counter > 0 {
				e.counter--
			} else if e.apuLengthCounter.lengthCounterHalt {
				e.counter = 15
			}
		}
	} else {
		e.start = false
		e.counter = 15
		e.divider = int8(e.volume)
	}
}

// APU
type APU struct {
	console       *Console
	channel       chan float32
	sampleRate    float64
	frameCounter  *FrameCounter
	square1       *SquareChannel
	square2       *SquareChannel
	triangle      *TriangleChannel
	noise         *NoiseChannel
	dmc           *DeltaModulationChannel
	currentCycle  uint64
	previousCycle uint64
	framePeriod   byte
	frameValue    byte
	needToRun     bool
	frameIRQ      bool
	filterChain   APUFilterChain
}

func NewAPU(console *Console) *APU {
	apu := APU{}
	apu.console = console
	apu.frameCounter = NewFrameCounter(console)
	apu.square1 = NewSquareChannel(console, true)
	apu.square2 = NewSquareChannel(console, false)
	apu.triangle = NewTriangleChannel(console)
	apu.noise = NewNoiseChannel(console)
	apu.dmc = NewDeltaModulationChannel(console)
	return &apu
}

func (apu *APU) Reset() {
	apu.currentCycle = 0
	apu.previousCycle = 0
	apu.square1.Reset()
	apu.square2.Reset()
	apu.triangle.Reset()
	apu.noise.Reset()
	apu.dmc.Reset()
}

func (apu *APU) SetNeedToRun() {
	apu.needToRun = true
}

func (apu *APU) NeedToRun(currentCycle uint32) bool {
	if apu.dmc.NeedToRun() || apu.needToRun {
		apu.needToRun = false
		return true
	}

	cyclesToRun := currentCycle - uint32(apu.previousCycle)
	return apu.frameCounter.NeedToRun(cyclesToRun) || apu.dmc.IRQPending(cyclesToRun)
}

func (apu *APU) Run() {
	cyclesToRun := int32(apu.currentCycle - apu.previousCycle)

	for cyclesToRun > 0 {
		apu.previousCycle += uint64(apu.frameCounter.Run(&cyclesToRun))

		apu.square1.apuEnvelope.apuLengthCounter.ReloadCounter()
		apu.square2.apuEnvelope.apuLengthCounter.ReloadCounter()
		apu.noise.apuEnvelope.apuLengthCounter.ReloadCounter()
		apu.triangle.apuLengthCounter.ReloadCounter()

		apu.square1.Run(uint32(apu.previousCycle))
		apu.square2.Run(uint32(apu.previousCycle))
		apu.noise.Run(uint32(apu.previousCycle))
		apu.triangle.Run(uint32(apu.previousCycle))
		apu.dmc.Run(uint32(apu.previousCycle))
	}
}

func (apu *APU) EndFrame() {
	apu.Run()
	apu.square1.apuEnvelope.apuLengthCounter.baseAPUChannel.EndFrame()
	apu.square2.apuEnvelope.apuLengthCounter.baseAPUChannel.EndFrame()
	apu.triangle.apuLengthCounter.baseAPUChannel.EndFrame()
	apu.noise.apuEnvelope.apuLengthCounter.baseAPUChannel.EndFrame()
	apu.dmc.baseAPUChannel.EndFrame()

	apu.currentCycle = 0
	apu.previousCycle = 0
}

func (apu *APU) FrameCounterTick(frameType FrameType) {
	apu.square1.apuEnvelope.TickEnvelope()
	apu.square2.apuEnvelope.TickEnvelope()
	apu.triangle.TickLinearCounter()
	apu.noise.apuEnvelope.TickEnvelope()

	if frameType == FRAME_TYPE_HALF_FRAME {
		apu.square1.apuEnvelope.apuLengthCounter.TickLengthCounter()
		apu.square2.apuEnvelope.apuLengthCounter.TickLengthCounter()
		apu.triangle.apuLengthCounter.TickLengthCounter()
		apu.noise.apuEnvelope.apuLengthCounter.TickLengthCounter()

		apu.square1.TickSweep()
		apu.square2.TickSweep()
	}
}

func (apu *APU) Step() {
	cycle1 := apu.currentCycle
	apu.currentCycle++
	cycle2 := apu.currentCycle

	// XXX: Need apu.NeedToRun?
	// XXX: Magic Number
	// if apu.currentCycle == 10000-1 {
	// 	apu.EndFrame()
	// } else if apu.NeedToRun(uint32(apu.currentCycle)) {
	// 	apu.Run()
	// }

	apu.Run()

	// refs: github.com/fogleman/nes/nes/apu.go
	s1 := int(float64(cycle1) / apu.sampleRate)
	s2 := int(float64(cycle2) / apu.sampleRate)
	if s1 != s2 {
		apu.sendSample()
	}
}

func (apu *APU) sendSample() {
	output := apu.filterChain.Step(apu.output())
	select {
	case apu.channel <- output:
	default:
	}
}

func (apu *APU) output() float32 {
	p1 := apu.square1.currentOutput
	p2 := apu.square2.currentOutput
	t := apu.triangle.currentOutput
	n := apu.noise.currentOutput
	d := apu.dmc.currentOutput
	pulseOut := squareTable[p1+p2]
	tndOut := tndTable[3*t+2*n+d]
	return (pulseOut + tndOut)
}

type NoiseInfo struct {
	Out    byte
	Period uint16
}

type DMCInfo struct {
	Out    byte
	Period uint16
}

type APUCurrentInfo struct {
	Square1  float32
	Square2  float32
	Triangle float32
	Noise    *NoiseInfo
	DMC      *DMCInfo
}

func (apu *APU) CurrentInfo() *APUCurrentInfo {
	var s1 float32 = 0
	if apu.square1.realPeriod != 0 {
		s1 = float32(CPUFrequency) / (16.0 * (float32(apu.square1.realPeriod) + 1))
	}
	var s2 float32 = 0
	if apu.square2.realPeriod != 0 {
		s2 = float32(CPUFrequency) / (16.0 * (float32(apu.square2.realPeriod) + 1))
	}
	var t float32 = 0
	if apu.triangle.apuLengthCounter.baseAPUChannel.period != 0 {
		t = float32(CPUFrequency) / (16.0 * (float32(apu.triangle.apuLengthCounter.baseAPUChannel.period) + 1))
	}

	n := &NoiseInfo{}
	n.Out = apu.noise.currentOutput
	n.Period = apu.noise.apuEnvelope.apuLengthCounter.baseAPUChannel.period

	d := &DMCInfo{}
	d.Out = apu.dmc.currentOutput
	d.Period = apu.dmc.baseAPUChannel.period

	return &APUCurrentInfo{
		Square1:  s1,
		Square2:  s2,
		Triangle: t,
		Noise:    n,
		DMC:      d,
	}
}

func (apu *APU) readRegister(address uint16) byte {
	var status byte
	switch address {
	case 0x4015:
		status = apu.readStatus()
		// default:
		// 	log.Fatalf("unhandled apu register read at address: 0x%04X", address)
	}

	apu.console.CPU.ClearIRQSource(IRQ_FRAME_COUNTER)
	return status
}

func (apu *APU) writeRegister(address uint16, value byte) {
	switch address {
	case 0x4000:
		apu.square1.WriteRAM(address, value)
	case 0x4001:
		apu.square1.WriteRAM(address, value)
	case 0x4002:
		apu.square1.WriteRAM(address, value)
	case 0x4003:
		apu.square1.WriteRAM(address, value)
	case 0x4004:
		apu.square2.WriteRAM(address, value)
	case 0x4005:
		apu.square2.WriteRAM(address, value)
	case 0x4006:
		apu.square2.WriteRAM(address, value)
	case 0x4007:
		apu.square2.WriteRAM(address, value)
	case 0x4008:
		apu.triangle.WriteRAM(address, value)
	case 0x400A:
		apu.triangle.WriteRAM(address, value)
	case 0x400B:
		apu.triangle.WriteRAM(address, value)
	case 0x400C:
		apu.noise.WriteRAM(address, value)
	case 0x400D:
		// NOTHING DONE
	case 0x400E:
		apu.noise.WriteRAM(address, value)
	case 0x400F:
		apu.noise.WriteRAM(address, value)
	case 0x4010:
		apu.dmc.WriteRAM(address, value)
	case 0x4011:
		apu.dmc.WriteRAM(address, value)
	case 0x4012:
		apu.dmc.WriteRAM(address, value)
	case 0x4013:
		apu.dmc.WriteRAM(address, value)
	case 0x4015:
		apu.WriteRAM(address, value)
	case 0x4017:
		apu.frameCounter.WriteRAM(address, value)
		// default:
		// 	log.Fatalf("unhandled apu register write at address: 0x%04X", address)
	}
}

func (apu *APU) readStatus() byte {
	apu.Run()

	var status byte
	if apu.square1.apuEnvelope.apuLengthCounter.GetStatus() {
		status |= 0x01
	}
	if apu.square2.apuEnvelope.apuLengthCounter.GetStatus() {
		status |= 0x02
	}
	if apu.triangle.apuLengthCounter.GetStatus() {
		status |= 0x04
	}
	if apu.noise.apuEnvelope.apuLengthCounter.GetStatus() {
		status |= 0x08
	}
	if apu.dmc.GetStatus() {
		status |= 0x10
	}
	if apu.console.CPU.HasIRQSource(IRQ_FRAME_COUNTER) {
		status |= 0x40
	}
	if apu.console.CPU.HasIRQSource(IRQ_DMC) {
		status |= 0x80
	}

	return status
}

func (apu *APU) WriteRAM(address uint16, value byte) {
	apu.Run()

	apu.console.CPU.ClearIRQSource(IRQ_DMC)

	apu.square1.apuEnvelope.apuLengthCounter.SetEnabled((value & 0x01) == 0x01)
	apu.square2.apuEnvelope.apuLengthCounter.SetEnabled((value & 0x02) == 0x02)
	apu.triangle.apuLengthCounter.SetEnabled((value & 0x04) == 0x04)
	apu.noise.apuEnvelope.apuLengthCounter.SetEnabled((value & 0x08) == 0x08)
	apu.dmc.SetEnabled((value & 0x10) == 0x10)
}

func (apu *APU) GetDMCReadAddress() uint16 {
	return apu.dmc.GetDMCReadAddress()
}

func (apu *APU) SetDMCReadBuffer(value byte) {
	apu.dmc.SetDMCReadBuffer(value)
}

// FrameCounter

type FrameType byte

const (
	FRAME_TYPE_NONE          FrameType = 0
	FRAME_TYPE_QUARTER_FRAME FrameType = 1
	FRAME_TYPE_HALF_FRAME    FrameType = 2
)

type FrameCounter struct {
	console *Console

	stepCycles            [2][6]int32
	frameType             [2][6]FrameType
	previousCycle         int32
	currentStep           uint32
	stepMode              uint32
	inhibitIRQ            bool
	blockFrameCounterTick uint8
	newValue              int16
	writeDelayCounter     int8
}

func NewFrameCounter(console *Console) *FrameCounter {
	// XXX: NTSC
	var stepCyclesTable [2][6]int32 = [2][6]int32{
		{7457, 14913, 22371, 29828, 29829, 29830},
		{7457, 14913, 22371, 29829, 37281, 37282},
	}
	var frameTypeTable [2][6]FrameType = [2][6]FrameType{
		{FRAME_TYPE_QUARTER_FRAME, FRAME_TYPE_HALF_FRAME, FRAME_TYPE_QUARTER_FRAME, FRAME_TYPE_NONE, FRAME_TYPE_HALF_FRAME, FRAME_TYPE_NONE},
		{FRAME_TYPE_QUARTER_FRAME, FRAME_TYPE_HALF_FRAME, FRAME_TYPE_QUARTER_FRAME, FRAME_TYPE_NONE, FRAME_TYPE_HALF_FRAME, FRAME_TYPE_NONE},
	}

	f := &FrameCounter{
		console: console,
	}

	for i := 0; i < len(f.stepCycles); i++ {
		for j := 0; j < len(f.stepCycles[0]); j++ {
			f.stepCycles[i][j] = stepCyclesTable[i][j]
		}
	}

	for i := 0; i < len(f.frameType); i++ {
		for j := 0; j < len(f.frameType[0]); j++ {
			f.frameType[i][j] = frameTypeTable[i][j]
		}
	}

	return f
}

func (f *FrameCounter) Reset() {
	f.previousCycle = 0

	f.stepMode = 0

	f.currentStep = 0

	f.newValue = 0
	f.writeDelayCounter = 3
	f.inhibitIRQ = false

	f.blockFrameCounterTick = 0
}

func (f *FrameCounter) Run(cyclesToRun *int32) uint32 {
	var cyclesRan uint32

	if f.previousCycle+*cyclesToRun >= f.stepCycles[f.stepMode][f.currentStep] {
		if f.inhibitIRQ == false && f.stepMode == 0 && f.currentStep >= 3 {
			f.console.CPU.SetIRQSource(IRQ_FRAME_COUNTER)
		}

		t := f.frameType[f.stepMode][f.currentStep]
		if t != FRAME_TYPE_NONE && f.blockFrameCounterTick == 0 {
			f.console.APU.FrameCounterTick(t)

			f.blockFrameCounterTick = 2
		}

		if f.stepCycles[f.stepMode][f.currentStep] < f.previousCycle {
			cyclesRan = 0
		} else {
			cyclesRan = uint32(f.stepCycles[f.stepMode][f.currentStep] - f.previousCycle)
		}

		cyclesRan -= cyclesRan

		f.currentStep++
		if f.currentStep == 6 {
			f.currentStep = 0
			f.previousCycle = 0
		} else {
			f.previousCycle += int32(cyclesRan)
		}
	} else {
		cyclesRan = uint32(*cyclesToRun)
		*cyclesToRun = 0
		f.previousCycle += int32(cyclesRan)
	}

	if f.newValue >= 0 {
		f.writeDelayCounter--
		if f.writeDelayCounter == 0 {
			if (f.newValue & 0x80) == 0x80 {
				f.stepMode = 1
			} else {
				f.stepMode = 0
			}

			f.writeDelayCounter = -1
			f.currentStep = 0
			f.previousCycle = 0
			f.newValue = -1

			if f.stepMode > 0 && f.blockFrameCounterTick == 0 {
				f.console.APU.FrameCounterTick(FRAME_TYPE_HALF_FRAME)
				f.blockFrameCounterTick = 2
			}
		}
	}

	if f.blockFrameCounterTick > 0 {
		f.blockFrameCounterTick--
	}

	return cyclesRan
}

func (f *FrameCounter) NeedToRun(cyclesToRun uint32) bool {
	return f.newValue >= 0 || f.blockFrameCounterTick > 0 || (f.previousCycle+int32(cyclesToRun) >= f.stepCycles[f.stepMode][f.currentStep]-1)
}

func (f *FrameCounter) ReadRAM(addr uint16) byte {
	return 0
}

func (f *FrameCounter) WriteRAM(addr uint16, value byte) {
	f.console.APU.Run()
	f.newValue = int16(value)

	if (f.console.CPU.cycleCount & 0x01) > 0 {
		f.writeDelayCounter = 4
	} else {
		f.writeDelayCounter = 3
	}

	f.inhibitIRQ = (value & 0x40) == 0x40
	if f.inhibitIRQ {
		f.console.CPU.ClearIRQSource(IRQ_FRAME_COUNTER)
	}
}

// Square
type SquareChannel struct {
	console     *Console
	apuEnvelope *APUEnvelope

	dutySequences [4][8]byte
	isChannel1    bool
	isMMC5Square  bool

	duty    byte
	dutyPos byte

	sweepEnabled      bool
	sweepPeriod       byte
	sweepNegate       bool
	sweepShift        byte
	reloadSweep       bool
	sweepDivider      byte
	sweepTargetPeriod uint32
	realPeriod        uint16

	currentOutput byte
}

func NewSquareChannel(console *Console, isChannel1 bool) *SquareChannel {
	// XXX: NTSC
	var dutyTable = [][]byte{
		{0, 1, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 0, 0, 0, 0, 0},
		{0, 1, 1, 1, 1, 0, 0, 0},
		{1, 0, 0, 1, 1, 1, 1, 1},
	}

	s := &SquareChannel{
		console:           console,
		apuEnvelope:       NewAPUEnvelope(console),
		isChannel1:        isChannel1,
		isMMC5Square:      false,
		duty:              0,
		dutyPos:           0,
		sweepEnabled:      false,
		sweepPeriod:       0,
		sweepNegate:       false,
		sweepShift:        0,
		reloadSweep:       false,
		sweepDivider:      0,
		sweepTargetPeriod: 0,
		realPeriod:        0,
	}

	for i := 0; i < len(s.dutySequences); i++ {
		for j := 0; j < len(s.dutySequences[i]); j++ {
			s.dutySequences[i][j] = dutyTable[i][j]
		}
	}

	return s
}

func (s *SquareChannel) IsMuted() bool {
	return s.realPeriod < 8 || (s.sweepNegate == false && s.sweepTargetPeriod > 0x7FF)
}

func (s *SquareChannel) InitializeSweep(regValue byte) {
	s.sweepEnabled = (regValue & 0x80) == 0x80
	s.sweepNegate = (regValue & 0x08) == 0x08

	s.sweepPeriod = ((regValue & 0x70) >> 4) + 1
	s.sweepShift = (regValue & 0x07)

	s.UpdateTargetPeriod()

	s.reloadSweep = true
}

func (s *SquareChannel) UpdateTargetPeriod() {
	shiftResult := (s.realPeriod >> uint16(s.sweepShift))
	if s.sweepNegate {
		s.sweepTargetPeriod = uint32(s.realPeriod) - uint32(shiftResult)
		if s.isChannel1 {
			s.sweepTargetPeriod--
		}
	} else {
		s.sweepTargetPeriod = uint32(s.realPeriod) + uint32(shiftResult)
	}
}

func (s *SquareChannel) SetPeriod(newPeriod uint16) {
	s.realPeriod = newPeriod
	s.apuEnvelope.apuLengthCounter.baseAPUChannel.period = (s.realPeriod * 2) + 1
	s.UpdateTargetPeriod()
}

func (s *SquareChannel) Reset() {
	s.duty = 0
	s.dutyPos = 0

	s.realPeriod = 0

	s.sweepEnabled = false
	s.sweepPeriod = 0
	s.sweepNegate = false
	s.sweepShift = 0
	s.reloadSweep = false
	s.sweepDivider = 0
	s.sweepTargetPeriod = 0
	s.UpdateTargetPeriod()
}

func (s *SquareChannel) WriteRAM(addr uint16, value byte) {
	s.console.APU.Run()

	switch addr & 0x03 {
	case 0:
		// 4000 & 4004
		s.apuEnvelope.apuLengthCounter.InitializeLengthCounter(value&0x20 == 0x20)
		s.apuEnvelope.InitializeEnvelope(value)

		s.duty = (value & 0xC0) >> 6
		// XXX:
		// if(_console->GetSettings()->CheckFlag(EmulationFlags::SwapDutyCycles)) {
		// 	_duty = ((_duty & 0x02) >> 1) | ((_duty & 0x01) << 1);
		// }
	case 1:
		// 4001 & 4005
		s.InitializeSweep(value)
	case 2:
		// 4002 & 4006
		s.SetPeriod((s.realPeriod & 0x0700) | uint16(value))
	case 3:
		// 4003 & 4007
		s.apuEnvelope.apuLengthCounter.LoadLengthCounter(value >> 3)
		s.SetPeriod((s.realPeriod & 0xFF) | ((uint16(value) & 0x07) << 8))

		s.dutyPos = 0

		s.apuEnvelope.ResetEnvelope()
	}

	if s.isMMC5Square == false {
		s.UpdateOutput()
	}
}

func (s *SquareChannel) TickSweep() {
	s.sweepDivider--
	if s.sweepDivider == 0 {
		if s.sweepShift > 0 && s.sweepEnabled && s.realPeriod >= 8 && s.sweepTargetPeriod <= 0x7FF {
			s.SetPeriod(uint16(s.sweepTargetPeriod))
		}
		s.sweepDivider = s.sweepPeriod
	}

	if s.reloadSweep {
		s.sweepDivider = s.sweepPeriod
		s.reloadSweep = false
	}
}

func (s *SquareChannel) UpdateOutput() {
	if s.IsMuted() {
		s.currentOutput = 0
	} else {
		s.currentOutput = s.dutySequences[s.duty][s.dutyPos] * byte(s.apuEnvelope.GetVolume())
	}
}

func (s *SquareChannel) Run(targetCycle uint32) {
	cyclesToRun := int32(targetCycle - s.apuEnvelope.apuLengthCounter.baseAPUChannel.previousCycle)
	for cyclesToRun > int32(s.apuEnvelope.apuLengthCounter.baseAPUChannel.timer) {
		cyclesToRun -= int32(s.apuEnvelope.apuLengthCounter.baseAPUChannel.timer) + 1
		s.apuEnvelope.apuLengthCounter.baseAPUChannel.previousCycle += uint32(s.apuEnvelope.apuLengthCounter.baseAPUChannel.timer) + 1
		// Begin Clock
		s.dutyPos = (s.dutyPos - 1) & 0x07
		s.UpdateOutput()
		// End Clock
		s.apuEnvelope.apuLengthCounter.baseAPUChannel.timer = s.apuEnvelope.apuLengthCounter.baseAPUChannel.period
	}

	s.apuEnvelope.apuLengthCounter.baseAPUChannel.timer -= uint16(cyclesToRun)
	s.apuEnvelope.apuLengthCounter.baseAPUChannel.previousCycle = targetCycle
}

// Triangle

type TriangleChannel struct {
	console          *Console
	apuLengthCounter *APULengthCounter

	sequence            [32]byte
	linearCounter       byte
	linearCounterReload byte
	linearReloadFlag    bool
	linearControlFlag   bool
	sequencePosition    byte

	currentOutput byte
}

func NewTriangleChannel(console *Console) *TriangleChannel {
	// XXX: NTSC
	var triangleTable = []byte{
		15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
	}

	t := &TriangleChannel{
		console:             console,
		apuLengthCounter:    NewAPULengthCounter(console),
		linearCounter:       0,
		linearCounterReload: 0,
		linearReloadFlag:    false,
		linearControlFlag:   false,
	}

	for i := 0; i < len(t.sequence); i++ {
		t.sequence[i] = triangleTable[i]
	}

	return t
}

func (t *TriangleChannel) Reset() {
	t.apuLengthCounter.Reset()

	t.linearCounter = 0
	t.linearCounterReload = 0
	t.linearReloadFlag = false
	t.linearControlFlag = false

	t.sequencePosition = 0
}

func (t *TriangleChannel) TickLinearCounter() {
	if t.linearReloadFlag {
		t.linearCounter = t.linearCounterReload
	} else if t.linearCounter > 0 {
		t.linearCounter--
	}

	if t.linearControlFlag == false {
		t.linearReloadFlag = false
	}
}

func (t *TriangleChannel) WriteRAM(address uint16, value byte) {
	t.console.APU.Run()

	switch address & 0x03 {
	case 0:
		// 4008
		t.linearControlFlag = (value & 0x80) == 0x80
		t.linearCounterReload = value & 0x7F

		t.apuLengthCounter.InitializeLengthCounter(t.linearControlFlag)
	case 2:
		// 400A
		t.apuLengthCounter.baseAPUChannel.period &= ^uint16(0x00FF)
		t.apuLengthCounter.baseAPUChannel.period |= uint16(value)
	case 3:
		// 400B
		t.apuLengthCounter.LoadLengthCounter(value >> 3)

		t.apuLengthCounter.baseAPUChannel.period &= ^uint16(0xFF00)
		t.apuLengthCounter.baseAPUChannel.period |= (uint16(value) & 0x07) << 8

		t.linearReloadFlag = true
	}
}

func (t *TriangleChannel) Run(targetCycle uint32) {
	cyclesToRun := int32(targetCycle - t.apuLengthCounter.baseAPUChannel.previousCycle)
	for cyclesToRun > int32(t.apuLengthCounter.baseAPUChannel.timer) {
		cyclesToRun -= int32(t.apuLengthCounter.baseAPUChannel.timer) + 1
		t.apuLengthCounter.baseAPUChannel.previousCycle += uint32(t.apuLengthCounter.baseAPUChannel.timer) + 1
		// Begin Clock
		if t.apuLengthCounter.lengthCounter > 0 && t.linearCounter > 0 {
			t.sequencePosition = (t.sequencePosition + 1) & 0x1F

			// if(_period >= 2 || !_console->GetSettings()->CheckFlag(EmulationFlags::SilenceTriangleHighFreq)) {
			// 	//Disabling the triangle channel when period is < 2 removes "pops" in the audio that are caused by the ultrasonic frequencies
			// 	//This is less "accurate" in terms of emulation, so this is an option (disabled by default)
			// 	AddOutput(_sequence[_sequencePosition]);
			// }
			if t.apuLengthCounter.baseAPUChannel.period >= 2 {
				t.currentOutput = t.sequence[t.sequencePosition]
			}
		}
		// End Clock
		t.apuLengthCounter.baseAPUChannel.timer = t.apuLengthCounter.baseAPUChannel.period
	}

	t.apuLengthCounter.baseAPUChannel.timer -= uint16(cyclesToRun)
	t.apuLengthCounter.baseAPUChannel.previousCycle = targetCycle
}

// Noise

type NoiseChannel struct {
	console     *Console
	apuEnvelope *APUEnvelope

	noisePeriodLookupTable [16]uint16
	shiftRegister          uint16
	modeFlag               bool

	currentOutput byte
}

func NewNoiseChannel(console *Console) *NoiseChannel {
	// XXX: NTSC
	var noiseTable = []uint16{
		4, 8, 16, 32, 64, 96, 128, 160, 202, 254, 380, 508, 762, 1016, 2034, 4068,
	}

	n := &NoiseChannel{
		console:       console,
		apuEnvelope:   NewAPUEnvelope(console),
		shiftRegister: 1,
		modeFlag:      false,
	}

	for i := 0; i < len(n.noisePeriodLookupTable); i++ {
		n.noisePeriodLookupTable[i] = noiseTable[i]
	}

	return n
}

func (n *NoiseChannel) Reset() {
	n.apuEnvelope.Reset()

	n.apuEnvelope.apuLengthCounter.baseAPUChannel.period = n.noisePeriodLookupTable[0] - 1
	n.shiftRegister = 1
	n.modeFlag = false
}

func (n *NoiseChannel) WriteRAM(addr uint16, value uint8) {
	n.console.APU.Run()

	switch addr & 0x03 {
	case 0:
		// 400C
		n.apuEnvelope.apuLengthCounter.InitializeLengthCounter((value & 0x20) == 0x20)
		n.apuEnvelope.InitializeEnvelope(value)
	case 2:
		// 400E
		n.apuEnvelope.apuLengthCounter.baseAPUChannel.period = n.noisePeriodLookupTable[value&0x0F] - 1
		n.modeFlag = (value & 0x80) == 0x80
	case 3:
		// 400F
		n.apuEnvelope.apuLengthCounter.LoadLengthCounter(value >> 3)

		n.apuEnvelope.ResetEnvelope()
	}
}

func (n *NoiseChannel) Run(targetCycle uint32) {
	cyclesToRun := int32(targetCycle - n.apuEnvelope.apuLengthCounter.baseAPUChannel.previousCycle)
	for cyclesToRun > int32(n.apuEnvelope.apuLengthCounter.baseAPUChannel.timer) {
		cyclesToRun -= int32(n.apuEnvelope.apuLengthCounter.baseAPUChannel.timer) + 1
		n.apuEnvelope.apuLengthCounter.baseAPUChannel.previousCycle += uint32(n.apuEnvelope.apuLengthCounter.baseAPUChannel.timer) + 1
		// Begin Clock
		mode := n.modeFlag

		var v uint16
		if mode {
			v = 6
		} else {
			v = 1
		}
		feedback := (n.shiftRegister & 0x01) ^ ((n.shiftRegister >> v) & 0x01)
		n.shiftRegister >>= 1
		n.shiftRegister |= (feedback << 14)

		if (n.shiftRegister & 0x01) == 0x01 {
			n.currentOutput = 0
		} else {
			n.currentOutput = byte(n.apuEnvelope.GetVolume())
		}
		// End Clock
		n.apuEnvelope.apuLengthCounter.baseAPUChannel.timer = n.apuEnvelope.apuLengthCounter.baseAPUChannel.period
	}

	n.apuEnvelope.apuLengthCounter.baseAPUChannel.timer -= uint16(cyclesToRun)
	n.apuEnvelope.apuLengthCounter.baseAPUChannel.previousCycle = targetCycle
}

// DMC

type DeltaModulationChannel struct {
	console           *Console
	baseAPUChannel    *BaseAPUChannel
	periodLookupTable [16]uint16

	sampleAddr   uint16
	sampleLength uint16
	outputLevel  byte
	irqEnabled   bool
	loopFlag     bool

	currentAddr    uint16
	bytesRemaining uint16
	readBuffer     byte
	bufferEmpty    bool

	shiftRegister byte
	bitsRemaining byte
	silenceFlag   bool
	needToRun     bool
	needInit      byte

	lastValue4011 byte

	currentOutput byte
}

func NewDeltaModulationChannel(console *Console) *DeltaModulationChannel {
	// XXX: NTSC
	var dmcTable = [16]uint16{
		428, 380, 340, 320, 286, 254, 226, 214, 190, 160, 142, 128, 106, 84, 72, 54,
	}

	d := &DeltaModulationChannel{
		console:        console,
		baseAPUChannel: &BaseAPUChannel{},
		sampleAddr:     0,
		sampleLength:   0,
		outputLevel:    0,
		irqEnabled:     false,
		loopFlag:       false,
		currentAddr:    0,
		bytesRemaining: 0,
		readBuffer:     0,
		bufferEmpty:    true,
		shiftRegister:  0,
		bitsRemaining:  0,
		silenceFlag:    true,
		needToRun:      false,
		needInit:       0,
		lastValue4011:  0,
		currentOutput:  0,
	}

	for i := 0; i < len(d.periodLookupTable); i++ {
		d.periodLookupTable[i] = dmcTable[i]
	}

	return d
}

func (d *DeltaModulationChannel) Reset() {
	d.baseAPUChannel.Reset()

	d.sampleAddr = 0xC00
	d.sampleLength = 1

	d.outputLevel = 0
	d.irqEnabled = false
	d.loopFlag = false

	d.currentAddr = 0
	d.bytesRemaining = 0
	d.readBuffer = 0
	d.bufferEmpty = true

	d.shiftRegister = 0
	d.bitsRemaining = 8
	d.silenceFlag = true
	d.needToRun = false

	d.lastValue4011 = 0

	d.baseAPUChannel.period = d.periodLookupTable[0] - 1

	d.baseAPUChannel.timer = d.baseAPUChannel.period
}

func (d *DeltaModulationChannel) InitSample() {
	d.currentAddr = d.sampleAddr
	d.bytesRemaining = d.sampleLength
	d.needToRun = d.bytesRemaining > 0
}

func (d *DeltaModulationChannel) IRQPending(cyclesToRun uint32) bool {
	if d.irqEnabled && d.bytesRemaining > 0 {
		cyclesToEmptyBuffer := (uint32(d.bitsRemaining) + (uint32(d.bytesRemaining)+(uint32(d.bytesRemaining)-1)*8)*8)
		if cyclesToRun >= cyclesToEmptyBuffer {
			return true
		}
	}
	return false
}

func (d *DeltaModulationChannel) NeedToRun() bool {
	if d.needInit > 0 {
		d.needInit--
		if d.needInit == 0 {
			d.StartDMCTransfer()
		}
	}
	return d.needToRun
}

func (d *DeltaModulationChannel) GetStatus() bool {
	return d.bytesRemaining > 0
}

func (d *DeltaModulationChannel) WriteRAM(addr uint16, value byte) {
	d.console.APU.Run()

	switch addr & 0x03 {
	case 0:
		// 4010
		d.irqEnabled = (value & 0x80) == 0x80
		d.loopFlag = (value & 0x40) == 0x40

		d.baseAPUChannel.period = d.periodLookupTable[value&0x0F] - 1

		if d.irqEnabled == false {
			d.console.CPU.ClearIRQSource(IRQ_DMC)
		}
	case 1:
		// 4011
		newValue := value & 0x7F

		d.outputLevel = newValue

		// uint8_t previousLevel = _outputLevel;
		//
		// if(_console->GetSettings()->CheckFlag(EmulationFlags::ReduceDmcPopping) && abs(_outputLevel - previousLevel) > 50) {
		// 	//Reduce popping sounds for 4011 writes
		// 	_outputLevel -= (_outputLevel - previousLevel) / 2;
		// }

		d.currentOutput = d.outputLevel

		if d.lastValue4011 != value && newValue > 0 {
			d.console.SetNextFrameOverclockStatus(true)
		}

		d.lastValue4011 = newValue
	case 2:
		// 4012
		d.sampleAddr = uint16(0xC000 | (uint32(value) << 6))
		if value > 0 {
			d.console.SetNextFrameOverclockStatus(false)
		}
	case 3:
		// 4013
		d.sampleLength = (uint16(value<<4) | 0x0001)
		if value > 0 {
			d.console.SetNextFrameOverclockStatus(false)
		}
	}
}

func (d *DeltaModulationChannel) SetEnabled(enabled bool) {
	if enabled == false {
		d.bytesRemaining = 0
		d.needToRun = false
	} else if d.bytesRemaining == 0 {
		d.InitSample()

		if (d.console.CPU.cycleCount & 0x01) == 0 {
			d.needInit = 2
		} else {
			d.needInit = 3
		}
	}
}

func (d *DeltaModulationChannel) StartDMCTransfer() {
	if d.bufferEmpty && d.bytesRemaining > 0 {
		d.console.CPU.StartDMCTransfer()
	}
}

func (d *DeltaModulationChannel) GetDMCReadAddress() uint16 {
	return d.currentAddr
}

func (d *DeltaModulationChannel) SetDMCReadBuffer(value byte) {
	if d.bytesRemaining > 0 {
		d.readBuffer = value
		d.bufferEmpty = false

		d.currentAddr++
		if d.currentAddr == 0 {
			d.currentAddr = 0x8000
		}

		d.bytesRemaining--

		if d.bytesRemaining == 0 {
			d.needToRun = false
			if d.loopFlag {
				d.InitSample()
			} else if d.irqEnabled {
				d.console.CPU.SetIRQSource(IRQ_DMC)
			}
		}
	}
}

func (d *DeltaModulationChannel) Run(targetCycle uint32) {
	cyclesToRun := int32(targetCycle - d.baseAPUChannel.previousCycle)
	for cyclesToRun > int32(d.baseAPUChannel.timer) {
		cyclesToRun -= int32(d.baseAPUChannel.timer) + 1
		d.baseAPUChannel.previousCycle += uint32(d.baseAPUChannel.timer) + 1
		// Begin Clock
		d.Clock()
		// End Clock
		d.baseAPUChannel.timer = d.baseAPUChannel.period
	}

	d.baseAPUChannel.timer -= uint16(cyclesToRun)
	d.baseAPUChannel.previousCycle = targetCycle
}

func (d *DeltaModulationChannel) Clock() {
	if d.silenceFlag == false {
		if (d.shiftRegister & 0x01) > 0 {
			if d.outputLevel <= 125 {
				d.outputLevel += 2
			}
		} else {
			if d.outputLevel >= 2 {
				d.outputLevel -= 2
			}
		}
		d.shiftRegister >>= 1
	}

	d.bitsRemaining--
	if d.bitsRemaining == 0 {
		d.bitsRemaining = 8
		if d.bufferEmpty {
			d.silenceFlag = true
		} else {
			d.silenceFlag = false
			d.shiftRegister = d.readBuffer
			d.bufferEmpty = true
			d.StartDMCTransfer()
		}
	}

	d.currentOutput = d.outputLevel
}

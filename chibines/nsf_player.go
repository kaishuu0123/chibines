package chibines

import (
	"time"
)

type NSFPlayer struct {
	Console            *Console
	PlayCallInterval   float64
	LastPlayCall       time.Time
	CurrentSong        byte
	CurrentSongLen     time.Duration
	CurrentSongFadeLen time.Duration
	CurrentSongStart   time.Time

	NSFFileInfo *NSFFileInfo

	PlayState bool
}

func NewNSFPlayer(path string) (*NSFPlayer, error) {
	console, err := NewConsole(path, true)
	if err != nil {
		return nil, err
	}

	nsfFileInfo, err := ParseNSFFileInfo(path)
	if err != nil {
		return nil, err
	}

	nsf := &NSFPlayer{
		Console:          console,
		CurrentSong:      nsfFileInfo.StartingSong - 1,
		CurrentSongLen:   0,
		NSFFileInfo:      nsfFileInfo,
		PlayCallInterval: float64(nsfFileInfo.PlaySpeedNTSC) / 1000000.0,
		PlayState:        false,
	}

	nsf.initNSFtune(nsfFileInfo.StartingSong - 1)

	return nsf, nil
}

func (np *NSFPlayer) initNSFtune(songNum byte) {
	for addr := uint16(0x0000); addr < 0x0800; addr++ {
		np.Console.CPU.bus.WriteMemory(addr, 0x00)
	}

	for addr := uint16(0x6000); addr < 0x8000; addr++ {
		np.Console.CPU.bus.WriteMemory(addr, 0x00)
	}

	for addr := uint16(0x4000); addr < 0x4014; addr++ {
		np.Console.CPU.bus.WriteMemory(addr, 0x00)
	}
	np.Console.CPU.bus.WriteMemory(0x4015, 0x00)
	np.Console.CPU.bus.WriteMemory(0x4015, 0x0F)
	np.Console.CPU.bus.WriteMemory(0x4017, 0x40)

	if np.NSFFileInfo.usesBanks() {
		for i := uint16(0); i < 8; i++ {
			np.Console.CPU.bus.WriteMemory(0x5FF8+i, np.NSFFileInfo.BankSetup[i])
		}
	}

	np.Console.CPU.state.A = songNum
	np.Console.CPU.state.X = 0

	np.Console.CPU.state.SP = 0xFD
	np.Console.CPU.push16(0x0000)
	np.Console.CPU.state.PC = np.NSFFileInfo.InitAddress

	for np.Console.CPU.state.PC != 0x0001 {
		np.Console.CPU.Step()
	}

	np.CurrentSong = songNum
	np.CurrentSongLen = 0

	np.CurrentSongStart = time.Now()
	np.LastPlayCall = time.Now()
}

func (np *NSFPlayer) StepSeconds(seconds float64) {
	cycles := int(CPUFrequency * seconds)
	var now time.Time

	for cycles > 0 {
		now = time.Now()

		if np.Console.CPU.state.PC == 0x0001 {
			timeLeft := np.PlayCallInterval - now.Sub(np.LastPlayCall).Seconds()
			if timeLeft <= 0 {
				np.LastPlayCall = now
				np.Console.CPU.state.SP = 0xFD
				np.Console.CPU.push16(0x0000)
				np.Console.CPU.state.PC = np.NSFFileInfo.PlayAddress
			}
		}

		if np.Console.CPU.state.PC != 0x0001 {
			cycles -= np.Console.Step()
		} else {
			np.Console.CPU.StartCPUCycle(true)
			np.Console.CPU.EndCPUCycle(true)

			cycles -= 1
		}
	}
}

func (np *NSFPlayer) PrevSong() {
	switch {
	case np.CurrentSong == 0:
		np.CurrentSong = np.NSFFileInfo.TotalSongs - 1
	case np.CurrentSong > 0:
		np.CurrentSong--
	}
	np.initNSFtune(np.CurrentSong)
}

func (np *NSFPlayer) NextSong() {
	switch {
	case np.CurrentSong == np.NSFFileInfo.TotalSongs-1:
		np.CurrentSong = 0
	case np.CurrentSong < np.NSFFileInfo.TotalSongs-1:
		np.CurrentSong++
	}
	np.initNSFtune(np.CurrentSong)
}

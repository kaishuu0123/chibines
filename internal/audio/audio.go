package audio

import "github.com/gordonklaus/portaudio"

const GLOBAL_VOLUME = 0.5

type Audio struct {
	stream         *portaudio.Stream
	SampleRate     float64
	outputChannels int
	Channel        chan float32
}

func NewAudio() *Audio {
	a := Audio{}
	a.Channel = make(chan float32, 44100)
	return &a
}

func (a *Audio) Start() error {
	host, err := portaudio.DefaultHostApi()
	if err != nil {
		return err
	}
	parameters := portaudio.HighLatencyParameters(nil, host.DefaultOutputDevice)
	stream, err := portaudio.OpenStream(parameters, a.Callback)
	if err != nil {
		return err
	}
	if err := stream.Start(); err != nil {
		return err
	}
	a.stream = stream
	a.SampleRate = parameters.SampleRate
	a.outputChannels = parameters.Output.Channels
	return nil
}

func (a *Audio) Stop() error {
	return a.stream.Close()
}

func (a *Audio) Callback(out []float32) {
	var output float32
	for i := range out {
		if i%a.outputChannels == 0 {
			select {
			case sample := <-a.Channel:
				output = sample * GLOBAL_VOLUME
			default:
				output = 0
			}
		}
		out[i] = output
	}
}

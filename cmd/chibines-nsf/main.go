package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strings"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/gordonklaus/portaudio"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/kaishuu0123/chibines/chibines"
	"github.com/kaishuu0123/chibines/internal/audio"
	"github.com/kaishuu0123/chibines/internal/gui"
)

const (
	WINDOW_WIDTH  int = 640
	WINDOW_HEIGHT int = 480
)

var (
	windowFlags imgui.WindowFlags = imgui.WindowFlagsNoCollapse |
		imgui.WindowFlagsNoMove |
		imgui.WindowFlagsNoResize |
		imgui.WindowFlagsNoTitleBar
)

type NSFInfoForView struct {
	title     string
	artist    string
	copyright string
}

const FreqC0 = 32.7032

var NoteNames = []string{
	"C",
	"C#",
	"D",
	"D#",
	"E",
	"F",
	"F#",
	"G",
	"G#",
	"A",
	"A#",
	"B",
}

var window *gui.MasterWindow
var nsfPlayer *chibines.NSFPlayer
var audioForConsole *audio.Audio
var isRunning bool = false
var nsfInfoForView *NSFInfoForView

func NoteFromFreq(freq float64) float64 {
	return 12.0 * math.Log2(freq/FreqC0)
}

func NoteString(noteValue int) string {
	note := (noteValue - 1) % 12
	octave := (noteValue - 1) / 12

	return fmt.Sprintf("%s%d", NoteNames[note], octave)
}

func Freq2NoteString(freq float32) string {
	if freq < FreqC0 {
		return ""
	}

	noteFloat := NoteFromFreq(float64(freq))
	note := int(math.Round(noteFloat))
	// cents := math.Round((noteFloat - note) * 100.0)
	return NoteString(note + 1)
}

func StartAudio() {
	// initialize audio
	portaudio.Initialize()

	audioForConsole = audio.NewAudio()
	if err := audioForConsole.Start(); err != nil {
		log.Fatalln(err)
	}

	if nsfPlayer.Console == nil {
		log.Fatalln("console must be set")
	}

	nsfPlayer.Console.SetAudioChannel(audioForConsole.Channel)
	nsfPlayer.Console.SetAudioSampleRate(audioForConsole.SampleRate)
}

func StopAudio() {
	if isRunning {
		audioForConsole.Stop()
		nsfPlayer.Console.SetAudioSampleRate(0)
		nsfPlayer.Console.SetAudioChannel(nil)

		portaudio.Terminate()
	}
}

func ResetNSFPlayer(file_name string) {
	StopAudio()
	isRunning = false

	log.Println("Reset Console")
	log.Printf("NSF file path: %s\n", file_name)
	var err error
	nsfPlayer, err = chibines.NewNSFPlayer(file_name)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("TotalSongs: %d\n", nsfPlayer.NSFFileInfo.TotalSongs)
	log.Printf("SongName: %s\n", nsfPlayer.NSFFileInfo.SongName)
	log.Printf("ArtistName: %s\n", nsfPlayer.NSFFileInfo.ArtistName)
	log.Printf("Copyright: %s\n", nsfPlayer.NSFFileInfo.CopyrightHolder)

	nsfInfoForView = &NSFInfoForView{
		title:     fmt.Sprintf("%-9s : %s", "Title", nsfPlayer.NSFFileInfo.SongName),
		artist:    fmt.Sprintf("%-9s : %s", "Artist", nsfPlayer.NSFFileInfo.ArtistName),
		copyright: fmt.Sprintf("%-9s : %s", "Copyright", nsfPlayer.NSFFileInfo.CopyrightHolder),
	}

	isRunning = true

	StartAudio()
}

func onDrop(names []string) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s", names[0]))
	dropInFiles := sb.String()
	ResetNSFPlayer(dropInFiles)
}

func renderNSFPlayerGUI() {
	imgui.PushFont(window.FontsData[2])
	imgui.SetNextWindowPos(imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSize(imgui.Vec2{X: float32(WINDOW_WIDTH), Y: float32(WINDOW_HEIGHT)})

	// NSF File Information
	imgui.BeginV("Window", nil, windowFlags)
	imgui.Text(nsfInfoForView.title)
	imgui.Text(nsfInfoForView.artist)
	imgui.Text(nsfInfoForView.copyright)
	trackAndState := fmt.Sprintf("%-9s : %02d", "No", nsfPlayer.CurrentSong+1)
	imgui.Text(trackAndState)

	dl := imgui.WindowDrawList()
	pos := imgui.CursorPos()
	imgui.PushFont(window.FontsData[3])
	imgui.SetCursorPos(imgui.Vec2{X: pos.X, Y: pos.Y + 20})

	pos = imgui.CursorPos()
	freq := nsfPlayer.Console.APU.CurrentInfo()

	// Draw Visualizer (Keyboard & Noize & DMC)
	s1 := Freq2NoteString(freq.Square1)
	square1Text := fmt.Sprintf("Square 1: %s", s1)
	imgui.Text(square1Text)
	var s1KeyIndex int = -1
	if s1 != "" {
		s1KeyIndex = Pitch2KeyIndexTable[s1]
	}
	DrawPianoKeyboard(&dl, imgui.Vec2{X: pos.X + 0, Y: pos.Y + 25}, s1KeyIndex)
	imgui.SetCursorPos(imgui.Vec2{X: pos.X, Y: pos.Y + 80})

	pos = imgui.CursorPos()
	s2 := Freq2NoteString(freq.Square2)
	square2Text := fmt.Sprintf("Square 2: %s", s2)
	var s2KeyIndex int = -1
	if s2 != "" {
		s2KeyIndex = Pitch2KeyIndexTable[s2]
	}
	imgui.Text(square2Text)
	DrawPianoKeyboard(&dl, imgui.Vec2{X: pos.X + 0, Y: pos.Y + 25}, s2KeyIndex)
	imgui.SetCursorPos(imgui.Vec2{X: pos.X, Y: pos.Y + 80})

	pos = imgui.CursorPos()
	t := Freq2NoteString(freq.Triangle)
	triangleText := fmt.Sprintf("Triangle: %s", t)
	var triangleKeyIndex int = -1
	if t != "" {
		triangleKeyIndex = Pitch2KeyIndexTable[t]
	}
	imgui.Text(triangleText)
	DrawPianoKeyboard(&dl, imgui.Vec2{X: pos.X + 0, Y: pos.Y + 25}, triangleKeyIndex)
	imgui.SetCursorPos(imgui.Vec2{X: pos.X, Y: pos.Y + 80})

	pos = imgui.CursorPos()
	noiseText := fmt.Sprintf("Noise: Volume = %X Period = %d", freq.Noise.Out, freq.Noise.Period)
	imgui.Text(noiseText)
	dmcText := fmt.Sprintf("DMC  : Volume = %X Period = %d", freq.DMC.Out, freq.DMC.Period)
	imgui.Text(dmcText)

	// Status Line (Play Status & Help Text)
	pos = imgui.CursorPos()

	// XXX: For Debug
	// io := imgui.CurrentIO()
	// framerateText := fmt.Sprintf("Application average %.3f ms/frame (%.1f FPS)", 1000.0/io.Framerate(), io.Framerate())
	// imgui.Text(framerateText)

	var state string = "Stopped"
	if nsfPlayer.PlayState {
		state = "Playing"
	}
	stateText := fmt.Sprintf("State: %s", state)
	textSize := imgui.CalcTextSize(stateText, false, 0)
	pos = imgui.CursorPos()
	posX := pos.X
	posY := imgui.WindowHeight() - (textSize.Y * 2)
	imgui.SetCursorPos(imgui.Vec2{X: posX, Y: posY})
	imgui.Text(stateText)
	imgui.SameLine()
	helpText := fmt.Sprintf("Start/Stop = Enter | Prev = <- | Next = ->")
	textSize = imgui.CalcTextSize(helpText, false, 0)
	posX = imgui.WindowContentRegionWidth() - float32(textSize.X)
	pos = imgui.CursorPos()
	imgui.SetCursorPos(imgui.Vec2{X: posX, Y: pos.Y})
	imgui.Text(helpText)
	imgui.PopFont()

	imgui.End()
	imgui.PopFont()
}

func renderGUI(w *gui.MasterWindow) {
	w.Platform.NewFrame()
	imgui.NewFrame()

	if isRunning {
		renderNSFPlayerGUI()
	} else {
		var msg string = "ChibiNES NSF Player is currently stopped.\n\nPlease drag and drop NSF file."
		textSize := imgui.CalcTextSize(msg, false, 0)
		xpos := (float32(WINDOW_WIDTH) - textSize.X) / 2
		ypos := (float32(WINDOW_HEIGHT) - textSize.Y) / 2
		imgui.ForegroundDrawList().
			AddText(
				imgui.Vec2{X: xpos, Y: ypos},
				imgui.PackedColor(0xFFFFFFFF),
				msg,
			)
	}

	imgui.Render()

	w.Renderer.PreRender(w.ClearColor)
	w.Renderer.Render(w.Platform.DisplaySize(), w.Platform.FramebufferSize(), imgui.RenderedDrawData())
	w.Platform.PostRender()
}

func main() {
	flag.Parse()
	if len(flag.Args()) >= 1 {
		_, err := os.Stat(flag.Arg(0))
		if err != nil {
			log.Fatalln("no NSF file specified or found")
		}

		ResetNSFPlayer(flag.Arg(0))
	}
	defer StopAudio()

	window = gui.NewMasterWindow("ChibiNES NSF Player", WINDOW_WIDTH, WINDOW_HEIGHT, 0)
	window.SetDropCallback(onDrop)
	defer window.Renderer.Dispose()
	defer window.Platform.Dispose()

	style := imgui.CurrentStyle()
	style.SetColor(imgui.StyleColorWindowBg, imgui.Vec4{X: 0.0, Y: 0.0, Z: 0.0, W: 1.00})

	initPianoKeyboard()
	// Unlimit FPS
	glfw.SwapInterval(0)

	prev_timestamp := glfw.GetTime()

	previousKeyState := map[int]bool{
		int(glfw.KeyRight): false,
		int(glfw.KeyLeft):  false,
		int(glfw.KeyEnter): false,
	}

	glfwWindow := window.Platform.Window
	for !window.Platform.ShouldStop() {
		cur_timestamp := glfw.GetTime()

		window.Platform.ProcessEvents()

		if !previousKeyState[int(glfw.KeyRight)] && glfwWindow.GetKey(glfw.KeyRight) == glfw.Press {
			previousKeyState[int(glfw.KeyRight)] = true
			nsfPlayer.NextSong()
		}
		if previousKeyState[int(glfw.KeyRight)] && glfwWindow.GetKey(glfw.KeyRight) == glfw.Release {
			previousKeyState[int(glfw.KeyRight)] = false
		}
		if !previousKeyState[int(glfw.KeyLeft)] && glfwWindow.GetKey(glfw.KeyLeft) == glfw.Press {
			previousKeyState[int(glfw.KeyLeft)] = true
			nsfPlayer.PrevSong()
		}
		if previousKeyState[int(glfw.KeyLeft)] && glfwWindow.GetKey(glfw.KeyLeft) == glfw.Release {
			previousKeyState[int(glfw.KeyLeft)] = false
		}
		if !previousKeyState[int(glfw.KeyEnter)] && glfwWindow.GetKey(glfw.KeyEnter) == glfw.Press {
			previousKeyState[int(glfw.KeyEnter)] = true
			nsfPlayer.PlayState = !nsfPlayer.PlayState
		}
		if previousKeyState[int(glfw.KeyEnter)] && glfwWindow.GetKey(glfw.KeyEnter) == glfw.Release {
			previousKeyState[int(glfw.KeyEnter)] = false
		}

		dt := cur_timestamp - prev_timestamp
		prev_timestamp = cur_timestamp

		if isRunning && nsfPlayer.PlayState {
			nsfPlayer.StepSeconds(dt)
		}

		renderGUI(window)
	}
}

package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"strings"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/gordonklaus/portaudio"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/kaishuu0123/toynes/internal/audio"
	"github.com/kaishuu0123/toynes/internal/gui"
	"github.com/kaishuu0123/toynes/toynes"
	"golang.org/x/image/draw"
)

const (
	SCALE         int = 2
	WINDOW_WIDTH  int = 254 * SCALE
	WINDOW_HEIGHT int = 240 * SCALE
)

var (
	windowFlags imgui.WindowFlags = imgui.WindowFlagsNoCollapse |
		imgui.WindowFlagsNoMove |
		imgui.WindowFlagsNoResize |
		imgui.WindowFlagsHorizontalScrollbar
	consoleRun = false
)

var console *toynes.Console
var audioForConsole *audio.Audio

func StartAudio() {
	// initialize audio
	portaudio.Initialize()

	audioForConsole = audio.NewAudio()
	if err := audioForConsole.Start(); err != nil {
		log.Fatalln(err)
	}

	if console == nil {
		log.Fatalln("console must be set")
	}

	console.SetAudioChannel(audioForConsole.Channel)
	console.SetAudioSampleRate(audioForConsole.SampleRate)
}

func StopAudio() {
	if consoleRun {
		audioForConsole.Stop()
		console.SetAudioSampleRate(0)
		console.SetAudioChannel(nil)

		portaudio.Terminate()
	}
}

func ResetConsole(file_name string) {
	StopAudio()
	consoleRun = false

	log.Println("Reset Console")
	log.Printf("ROM file path: %s\n", file_name)
	var err error
	console, err = toynes.NewConsole(file_name, false)
	if err != nil {
		log.Fatalln(err)
	}
	consoleRun = true

	StartAudio()
}

func onDrop(names []string) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s", names[0]))
	dropInFiles := sb.String()
	ResetConsole(dropInFiles)
}

func renderGUI(w *gui.MasterWindow, texture *imgui.TextureID) {
	w.Platform.NewFrame()
	imgui.NewFrame()

	if consoleRun {
		imgui.BackgroundDrawList().
			AddImage(
				*texture,
				imgui.Vec2{X: 0, Y: 0},
				imgui.Vec2{X: float32(WINDOW_WIDTH), Y: float32(WINDOW_HEIGHT)},
			)
	} else {
		var msg string = "ToyNES is currently stopped.\n\nPlease drag and drop ROM file."
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
			log.Fatalln("no rom file specified or found")
		}

		ResetConsole(flag.Arg(0))
	}
	defer StopAudio()

	window := gui.NewMasterWindow("ToyNES", WINDOW_WIDTH, WINDOW_HEIGHT, 0)
	window.SetDropCallback(onDrop)
	screenImage := image.NewRGBA(image.Rect(0, 0, WINDOW_WIDTH*SCALE, WINDOW_HEIGHT*SCALE))

	var texture imgui.TextureID
	prev_timestamp := glfw.GetTime()

	var buffer *image.RGBA
	for !window.Platform.ShouldStop() {
		cur_timestamp := glfw.GetTime()
		window.Platform.ProcessEvents()

		if consoleRun {
			result1 := processInputController1(window.Platform.Window)
			console.SetButtons1(result1)
			result2 := processInputController2(window.Platform.Window)
			console.SetButtons2(result2)
		}

		dt := cur_timestamp - prev_timestamp
		prev_timestamp = cur_timestamp

		if consoleRun {
			console.StepSeconds(dt)

			buffer = console.Buffer()
			draw.NearestNeighbor.Scale(screenImage, screenImage.Bounds(), buffer, buffer.Bounds(), draw.Over, nil)
		}

		texture, _ = window.Renderer.CreateImageTexture(screenImage)
		renderGUI(window, &texture)
		window.Renderer.ReleaseImage(texture)
	}
}

func processInputController1(window *glfw.Window) [8]bool {
	var result [8]bool
	result[toynes.ButtonA] = window.GetKey(glfw.KeyZ) == glfw.Press
	result[toynes.ButtonB] = window.GetKey(glfw.KeyX) == glfw.Press
	result[toynes.ButtonSelect] = window.GetKey(glfw.KeyRightShift) == glfw.Press
	result[toynes.ButtonStart] = window.GetKey(glfw.KeyEnter) == glfw.Press
	result[toynes.ButtonUp] = window.GetKey(glfw.KeyUp) == glfw.Press
	result[toynes.ButtonDown] = window.GetKey(glfw.KeyDown) == glfw.Press
	result[toynes.ButtonLeft] = window.GetKey(glfw.KeyLeft) == glfw.Press
	result[toynes.ButtonRight] = window.GetKey(glfw.KeyRight) == glfw.Press
	return result
}

func processInputController2(window *glfw.Window) [8]bool {
	var result [8]bool
	result[toynes.ButtonA] = window.GetKey(glfw.KeyA) == glfw.Press
	result[toynes.ButtonB] = window.GetKey(glfw.KeyS) == glfw.Press
	result[toynes.ButtonSelect] = window.GetKey(glfw.KeyLeftShift) == glfw.Press
	result[toynes.ButtonStart] = window.GetKey(glfw.KeyE) == glfw.Press
	result[toynes.ButtonUp] = window.GetKey(glfw.KeyI) == glfw.Press
	result[toynes.ButtonDown] = window.GetKey(glfw.KeyK) == glfw.Press
	result[toynes.ButtonLeft] = window.GetKey(glfw.KeyJ) == glfw.Press
	result[toynes.ButtonRight] = window.GetKey(glfw.KeyL) == glfw.Press
	return result
}

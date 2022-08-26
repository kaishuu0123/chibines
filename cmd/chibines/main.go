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
	"github.com/kaishuu0123/chibines/chibines"
	"github.com/kaishuu0123/chibines/internal/audio"
	"github.com/kaishuu0123/chibines/internal/gui"
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
	isRunning = false
)

var console *chibines.Console
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
	if isRunning {
		audioForConsole.Stop()
		console.SetAudioSampleRate(0)
		console.SetAudioChannel(nil)

		portaudio.Terminate()
	}
}

func ResetConsole(file_name string) {
	StopAudio()
	isRunning = false

	log.Println("Reset Console")
	log.Printf("ROM file path: %s\n", file_name)
	var err error
	console, err = chibines.NewConsole(file_name, false)
	if err != nil {
		log.Fatalln(err)
	}
	isRunning = true

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

	if isRunning {
		imgui.BackgroundDrawList().
			AddImage(
				*texture,
				imgui.Vec2{X: 0, Y: 0},
				imgui.Vec2{X: float32(WINDOW_WIDTH), Y: float32(WINDOW_HEIGHT)},
			)
	} else {
		var msg string = "ChibiNES is currently stopped.\n\nPlease drag and drop ROM file."
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

	window := gui.NewMasterWindow("ChibiNES", WINDOW_WIDTH, WINDOW_HEIGHT, 0)
	window.SetDropCallback(onDrop)
	screenImage := image.NewRGBA(image.Rect(0, 0, WINDOW_WIDTH*SCALE, WINDOW_HEIGHT*SCALE))

	if glfw.Joystick1.Present() {
		joyname := glfw.Joystick1.GetName()
		log.Printf("Joystick1 name: %s\n", joyname)
	}

	var buffer *image.RGBA
	var texture imgui.TextureID
	prev_timestamp := glfw.GetTime()
	for !window.Platform.ShouldStop() {
		cur_timestamp := glfw.GetTime()
		window.Platform.ProcessEvents()

		if isRunning {
			result1 := processInputController1(window.Platform.Window)
			j1 := readJoyStick(glfw.Joystick1)
			console.SetButtons1(combineButtons(result1, j1))

			result2 := processInputController2(window.Platform.Window)
			console.SetButtons2(result2)
		}

		dt := cur_timestamp - prev_timestamp
		prev_timestamp = cur_timestamp

		if isRunning {
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
	result[chibines.ButtonA] = window.GetKey(glfw.KeyZ) == glfw.Press
	result[chibines.ButtonB] = window.GetKey(glfw.KeyX) == glfw.Press
	result[chibines.ButtonSelect] = window.GetKey(glfw.KeyRightShift) == glfw.Press
	result[chibines.ButtonStart] = window.GetKey(glfw.KeyEnter) == glfw.Press
	result[chibines.ButtonUp] = window.GetKey(glfw.KeyUp) == glfw.Press
	result[chibines.ButtonDown] = window.GetKey(glfw.KeyDown) == glfw.Press
	result[chibines.ButtonLeft] = window.GetKey(glfw.KeyLeft) == glfw.Press
	result[chibines.ButtonRight] = window.GetKey(glfw.KeyRight) == glfw.Press
	return result
}

func processInputController2(window *glfw.Window) [8]bool {
	var result [8]bool
	result[chibines.ButtonA] = window.GetKey(glfw.KeyA) == glfw.Press
	result[chibines.ButtonB] = window.GetKey(glfw.KeyS) == glfw.Press
	result[chibines.ButtonSelect] = window.GetKey(glfw.KeyLeftShift) == glfw.Press
	result[chibines.ButtonStart] = window.GetKey(glfw.KeyE) == glfw.Press
	result[chibines.ButtonUp] = window.GetKey(glfw.KeyI) == glfw.Press
	result[chibines.ButtonDown] = window.GetKey(glfw.KeyK) == glfw.Press
	result[chibines.ButtonLeft] = window.GetKey(glfw.KeyJ) == glfw.Press
	result[chibines.ButtonRight] = window.GetKey(glfw.KeyL) == glfw.Press
	return result
}

func readJoyStick(joy glfw.Joystick) [8]bool {
	var result [8]bool
	if !glfw.Joystick1.Present() {
		return result
	}
	joyname := glfw.Joystick1.GetName()
	axes := glfw.Joystick1.GetAxes()
	buttons := glfw.Joystick1.GetButtons()
	switch joyname {
	case "DUALSHOCK 4 Wireless Controller":
		result[chibines.ButtonA] = buttons[2] == 1
		result[chibines.ButtonB] = buttons[1] == 1
		result[chibines.ButtonSelect] = buttons[8] == 1
		result[chibines.ButtonStart] = buttons[9] == 1
		result[chibines.ButtonUp] = axes[1] < -0.5
		result[chibines.ButtonDown] = axes[1] > 0.5
		result[chibines.ButtonLeft] = axes[0] < -0.5
		result[chibines.ButtonRight] = axes[0] > 0.5
	default:
		result[chibines.ButtonA] = buttons[0] == 1
		result[chibines.ButtonB] = buttons[1] == 1
		result[chibines.ButtonSelect] = buttons[6] == 1
		result[chibines.ButtonStart] = buttons[7] == 1
		result[chibines.ButtonUp] = axes[1] < -0.5
		result[chibines.ButtonDown] = axes[1] > 0.5
		result[chibines.ButtonLeft] = axes[0] < -0.5
		result[chibines.ButtonRight] = axes[0] > 0.5
	}

	return result
}

func combineButtons(a, b [8]bool) [8]bool {
	var result [8]bool
	for i := 0; i < 8; i++ {
		result[i] = a[i] || b[i]
	}
	return result
}

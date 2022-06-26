package framework_for_imgui

import (
	"fmt"
	"math"
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/inkyblackness/imgui-go/v4"
)

// StringError describes a basic error with static information.
type StringError string

// Error returns the string itself.
func (err StringError) Error() string {
	return string(err)
}

const (
	ErrUnsupportedClientAPI = StringError("unsupported ClientAPI")
)

// GLFWClientAPI identifies the render system that shall be initialized.
type GLFWClientAPI string

// This is a list of GLFWClientAPI constants.
const (
	GLFWClientAPIOpenGL2 GLFWClientAPI = "OpenGL2"
	GLFWClientAPIOpenGL3 GLFWClientAPI = "OpenGL3"
)

// GLFW implements a platform based on github.com/go-gl/glfw (v3.2).
type GLFW struct {
	imguiIO imgui.IO

	Window *glfw.Window

	time             float64
	mouseJustPressed [3]bool
	onDropCallback   func([]string)
}

func init() {
	runtime.LockOSThread()
}

// NewGLFW attempts to initialize a GLFW context.
func NewGLFW(io imgui.IO, clientAPI GLFWClientAPI, width, height int, title string) (*GLFW, error) {
	err := glfw.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize glfw: %w", err)
	}

	switch clientAPI {
	case GLFWClientAPIOpenGL2:
		glfw.WindowHint(glfw.ContextVersionMajor, 2)
		glfw.WindowHint(glfw.ContextVersionMinor, 1)
	case GLFWClientAPIOpenGL3:
		glfw.WindowHint(glfw.ContextVersionMajor, 3)
		glfw.WindowHint(glfw.ContextVersionMinor, 2)
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
		glfw.WindowHint(glfw.OpenGLForwardCompatible, 1)
	default:
		glfw.Terminate()
		return nil, ErrUnsupportedClientAPI
	}

	window, err := glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		glfw.Terminate()
		return nil, fmt.Errorf("failed to create window: %w", err)
	}
	window.MakeContextCurrent()
	glfw.SwapInterval(1)

	platform := &GLFW{
		imguiIO: io,
		Window:  window,
	}
	platform.setKeyMapping()
	platform.installCallbacks()

	return platform, nil
}

// Dispose cleans up the resources.
func (platform *GLFW) Dispose() {
	platform.Window.Destroy()
	glfw.Terminate()
}

// ShouldStop returns true if the window is to be closed.
func (platform *GLFW) ShouldStop() bool {
	return platform.Window.ShouldClose()
}

// ProcessEvents handles all pending window events.
func (platform *GLFW) ProcessEvents() {
	glfw.PollEvents()
}

// DisplaySize returns the dimension of the display.
func (platform *GLFW) DisplaySize() [2]float32 {
	w, h := platform.Window.GetSize()
	return [2]float32{float32(w), float32(h)}
}

// FramebufferSize returns the dimension of the framebuffer.
func (platform *GLFW) FramebufferSize() [2]float32 {
	w, h := platform.Window.GetFramebufferSize()
	return [2]float32{float32(w), float32(h)}
}

// NewFrame marks the begin of a render pass. It forwards all current state to imgui IO.
func (platform *GLFW) NewFrame() {
	// Setup display size (every frame to accommodate for window resizing)
	displaySize := platform.DisplaySize()
	platform.imguiIO.SetDisplaySize(imgui.Vec2{X: displaySize[0], Y: displaySize[1]})

	// Setup time step
	currentTime := glfw.GetTime()
	if platform.time > 0 {
		platform.imguiIO.SetDeltaTime(float32(currentTime - platform.time))
	}
	platform.time = currentTime

	// Setup inputs
	if platform.Window.GetAttrib(glfw.Focused) != 0 {
		x, y := platform.Window.GetCursorPos()
		platform.imguiIO.SetMousePosition(imgui.Vec2{X: float32(x), Y: float32(y)})
	} else {
		platform.imguiIO.SetMousePosition(imgui.Vec2{X: -math.MaxFloat32, Y: -math.MaxFloat32})
	}

	for i := 0; i < len(platform.mouseJustPressed); i++ {
		down := platform.mouseJustPressed[i] || (platform.Window.GetMouseButton(glfwButtonIDByIndex[i]) == glfw.Press)
		platform.imguiIO.SetMouseButtonDown(i, down)
		platform.mouseJustPressed[i] = false
	}
}

// PostRender performs a buffer swap.
func (platform *GLFW) PostRender() {
	platform.Window.SwapBuffers()
}

func (platform *GLFW) setKeyMapping() {
	// Keyboard mapping. ImGui will use those indices to peek into the io.KeysDown[] array.
	platform.imguiIO.KeyMap(imgui.KeyTab, int(glfw.KeyTab))
	platform.imguiIO.KeyMap(imgui.KeyLeftArrow, int(glfw.KeyLeft))
	platform.imguiIO.KeyMap(imgui.KeyRightArrow, int(glfw.KeyRight))
	platform.imguiIO.KeyMap(imgui.KeyUpArrow, int(glfw.KeyUp))
	platform.imguiIO.KeyMap(imgui.KeyDownArrow, int(glfw.KeyDown))
	platform.imguiIO.KeyMap(imgui.KeyPageUp, int(glfw.KeyPageUp))
	platform.imguiIO.KeyMap(imgui.KeyPageDown, int(glfw.KeyPageDown))
	platform.imguiIO.KeyMap(imgui.KeyHome, int(glfw.KeyHome))
	platform.imguiIO.KeyMap(imgui.KeyEnd, int(glfw.KeyEnd))
	platform.imguiIO.KeyMap(imgui.KeyInsert, int(glfw.KeyInsert))
	platform.imguiIO.KeyMap(imgui.KeyDelete, int(glfw.KeyDelete))
	platform.imguiIO.KeyMap(imgui.KeyBackspace, int(glfw.KeyBackspace))
	platform.imguiIO.KeyMap(imgui.KeySpace, int(glfw.KeySpace))
	platform.imguiIO.KeyMap(imgui.KeyEnter, int(glfw.KeyEnter))
	platform.imguiIO.KeyMap(imgui.KeyEscape, int(glfw.KeyEscape))
	platform.imguiIO.KeyMap(imgui.KeyA, int(glfw.KeyA))
	platform.imguiIO.KeyMap(imgui.KeyC, int(glfw.KeyC))
	platform.imguiIO.KeyMap(imgui.KeyV, int(glfw.KeyV))
	platform.imguiIO.KeyMap(imgui.KeyX, int(glfw.KeyX))
	platform.imguiIO.KeyMap(imgui.KeyY, int(glfw.KeyY))
	platform.imguiIO.KeyMap(imgui.KeyZ, int(glfw.KeyZ))
}

func (platform *GLFW) installCallbacks() {
	platform.Window.SetMouseButtonCallback(platform.mouseButtonChange)
	platform.Window.SetScrollCallback(platform.mouseScrollChange)
	platform.Window.SetKeyCallback(platform.keyChange)
	platform.Window.SetCharCallback(platform.charChange)
	platform.Window.SetDropCallback(platform.onDrop)
}

var glfwButtonIndexByID = map[glfw.MouseButton]int{
	glfw.MouseButton1: mouseButtonPrimary,
	glfw.MouseButton2: mouseButtonSecondary,
	glfw.MouseButton3: mouseButtonTertiary,
}

var glfwButtonIDByIndex = map[int]glfw.MouseButton{
	mouseButtonPrimary:   glfw.MouseButton1,
	mouseButtonSecondary: glfw.MouseButton2,
	mouseButtonTertiary:  glfw.MouseButton3,
}

func (platform *GLFW) mouseButtonChange(window *glfw.Window, rawButton glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	buttonIndex, known := glfwButtonIndexByID[rawButton]

	if known && (action == glfw.Press) {
		platform.mouseJustPressed[buttonIndex] = true
	}
}

func (platform *GLFW) mouseScrollChange(window *glfw.Window, x, y float64) {
	platform.imguiIO.AddMouseWheelDelta(float32(x), float32(y))
}

func (platform *GLFW) keyChange(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action == glfw.Press {
		platform.imguiIO.KeyPress(int(key))
	}
	if action == glfw.Release {
		platform.imguiIO.KeyRelease(int(key))
	}

	// Modifiers are not reliable across systems
	platform.imguiIO.KeyCtrl(int(glfw.KeyLeftControl), int(glfw.KeyRightControl))
	platform.imguiIO.KeyShift(int(glfw.KeyLeftShift), int(glfw.KeyRightShift))
	platform.imguiIO.KeyAlt(int(glfw.KeyLeftAlt), int(glfw.KeyRightAlt))
	platform.imguiIO.KeySuper(int(glfw.KeyLeftSuper), int(glfw.KeyRightSuper))
}

func (platform *GLFW) charChange(window *glfw.Window, char rune) {
	platform.imguiIO.AddInputCharacters(string(char))
}

func (platform *GLFW) onDrop(window *glfw.Window, names []string) {
	window.Focus()

	if platform.onDropCallback != nil {
		platform.onDropCallback(names)
	}
}

// ClipboardText returns the current clipboard text, if available.
func (platform *GLFW) ClipboardText() string {
	return platform.Window.GetClipboardString()
}

// SetClipboardText sets the text as the current clipboard text.
func (platform *GLFW) SetClipboardText(text string) {
	platform.Window.SetClipboardString(text)
}

func (platform *GLFW) SetDropCallback(cb func(names []string)) {
	platform.onDropCallback = cb
}

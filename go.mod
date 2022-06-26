module github.com/kaishuu0123/toynes

go 1.18

require (
	github.com/edsrzf/mmap-go v1.1.0
	github.com/go-gl/gl v0.0.0-20211210172815-726fda9656d6
	github.com/go-gl/glfw/v3.3/glfw v0.0.0-20220622232848-a6c407ee30a0
	github.com/gordonklaus/portaudio v0.0.0-20220320131553-cc649ad523c1
	github.com/inkyblackness/imgui-go/v4 v4.4.0
	golang.org/x/image v0.0.0-20220413100746-70e8d0d3baa9
)

require (
	github.com/Code-Hex/dd v1.1.0 // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
)

replace github.com/inkyblackness/imgui-go/v4 => ./internal/kaishuu0123-imgui-go

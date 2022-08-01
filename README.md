# ChibiNES <!-- omit in toc -->

[![Godoc Reference](https://pkg.go.dev/badge/github.com/kaishuu0123/chibines)](https://pkg.go.dev/github.com/kaishuu0123/chibines)
[![GitHub Release](https://img.shields.io/github/v/release/kaishuu0123/chibines)](https://github.com/kaishuu0123/chibines/releases)
[![Github Actions Release Workflow](https://github.com/kaishuu0123/chibines/actions/workflows/release.yml/badge.svg)](https://github.com/kaishuu0123/chibines/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/kaishuu0123/chibines)](https://goreportcard.com/report/kaishuu0123/chibines)

ChibiNES is NES emulator written by Go. This is my favorite hobby project!

Porting [libretro/Mesen](https://github.com/libretro/Mesen/) to Go. Priority was given to mimicking Mesen's behavior rather than refactoring.

- [Screenshots](#screenshots)
  - [`cmd/chibines` (NES Console)](#cmdchibines-nes-console)
  - [`cmd/chibines-nsf` (NSF Player)](#cmdchibines-nsf-nsf-player)
- [Download](#download)
- [Spec](#spec)
- [Key binding](#key-binding)
- [Build & Run](#build--run)
- [Dependencies](#dependencies)
- [FAQ](#faq)
  - [Why do you only support these mappers?](#why-do-you-only-support-these-mappers)
- [ToDO](#todo)
- [Reference](#reference)
  - [Emulator](#emulator)
  - [Documents](#documents)
- [Credit](#credit)

## Screenshots

### `cmd/chibines` (NES Console)

![Screenshots](https://raw.github.com/kaishuu0123/chibines/main/screenshots/screenshots001.jpg)

### `cmd/chibines-nsf` (NSF Player)

https://user-images.githubusercontent.com/1567423/175847470-15224c46-65da-4cf0-87a8-ef876d67629f.mp4

## Download

- [See Release page](https://github.com/kaishuu0123/chibines/releases)

## Spec

- NTSC only
  - PAL, Dendy is not supported yet.
- Basic APU sound only (The following sound sources are currently not supported)
  - NAMCOT 16x (N160/N163)
  - MMC5
  - SUNSOFT 5B
  - VRC
- Mapper Support
  - [x] Mapper 0
  - [x] Mapper 1
  - [x] Mapper 2
  - [x] Mapper 3
  - [x] Mapper 4
  - [x] Mapper 16
  - [x] Mapper 31
    - For NSF Player

## Key binding

Player 1

|NES|Key|
|---|---|
| UP, DOWN, LEFT, RIGHT | Arrow Keys |
| Start | Enter |
| Select | Right Shift |
| A | Z |
| B | X |

Player 2

|NES|Key|
|---|---|
| UP, DOWN, LEFT, RIGHT | I, K, J, L |
| Start | E |
| Select | Left Shift |
| A | A |
| B | S |

## Build & Run

- Install Library
  - portaudio

MacOSX

```shell
brew install portaudio
```

- build

```shell
go build cmd/chibines/main.go
```

- or go run

```shell
go run cmd/chibines/main.go
```

## Dependencies

- Dear ImGUI ([inkyblackness/imgui-go](https://github.com/inkyblackness/imgui-go))
- GLFW
- portaudio

## FAQ

### Why do you only support these mappers?

Because it's my favorite games & for [nes-test-roms](https://github.com/christopherpow/nes-test-roms)

- Mapper0
  - [Super Mario Bros](https://nescartdb.com/profile/view/1486/)
- Mapper1
  - [Dragon Quest III](https://nescartdb.com/profile/view/1527/)
- Mapper16
  - [SD Gundam Gaiden: Knight Gundam Monogatari 2: Hikari no Kishi](https://nescartdb.com/profile/view/1752/)
  - [SD Gundam Gaiden: Knight Gundam Monogatari 3: Densetsu no Kishi Dan](https://nescartdb.com/profile/view/1753/)

## ToDO

- [X] CPU
- [X] PPU
- [X] APU
- [X] NSF Player (cmd/chibines-nsf)
  - like VirtuaNES
- [ ] 6502 compiler
  - like cc65
- [ ] disassembler
- [ ] Interpreter (cmd/chibines-interpreter)
- [ ] Sprite extractor (cmd/chibines-sprites)
- [ ] ROM info CLI (cmd/chibines-rominfo)
- [ ] Debugger (like [Mesen's Debugging tools](https://www.mesen.ca/docs/debugging.html))
- [ ] Test
  - [ ] [nes-test-roms](https://github.com/christopherpow/nes-test-roms/)
    - like [tetanes README.md](https://github.com/lukexor/tetanes)
  - [ ] go testing (like integration test)
    - [ ] CPU
    - [ ] PPU
    - [ ] APU

## Reference

### Emulator

- CPU, PPU, APU, Mapper etc ...
  - [libretro/Mesen](https://github.com/libretro/Mesen/)
  - [fogleman/nes](https://github.com/fogleman/nes)
  - [eteran/pretendo](https://github.com/eteran/pretendo)
  - [lukexor/tetanes](https://github.com/lukexor/tetanes)
  - [ivysnow/virtuanes](https://github.com/ivysnow/virtuanes/)
  - [sairoutine/faithjs](https://github.com/sairoutine/faithjs/)
- NSF Player
  - [theinternetftw/famigo](https://github.com/theinternetftw/famigo)

### Documents

- [Nesdev Wiki](https://www.nesdev.org/wiki/Nesdev_Wiki)
  - [CPU - NESdev Wiki](https://www.nesdev.org/wiki/CPU)
  - [PPU - NESdev Wiki](https://www.nesdev.org/wiki/PPU)
  - [APU - NESdev Wiki](https://www.nesdev.org/wiki/APU)
  - [Mapper - NESdev Wiki](https://www.nesdev.org/wiki/Mapper)
- [6502 Instruction Set](https://www.masswerk.at/6502/6502_instruction_set.html)
- [NesCartDB - Home](https://nescartdb.com/)

## Credit

- Font
  - [itouhiro/PixelMplus](https://github.com/itouhiro/PixelMplus)

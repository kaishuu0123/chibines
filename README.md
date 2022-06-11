# ToyNES <!-- omit in toc -->

ToyNES is NES emulator written by Go. This is my favorite hobby project!

Porting [libretro/Mesen](https://github.com/libretro/Mesen/) to Go. Priority was given to mimicking Mesen's behavior rather than refactoring.

- [Spec](#spec)
- [Key binding](#key-binding)
- [My ToDO list](#my-todo-list)
- [Build](#build)
- [Dependencies](#dependencies)
- [FAQ](#faq)
  - [Why do you only support these mappers?](#why-do-you-only-support-these-mappers)
- [Reference](#reference)
  - [Emulator](#emulator)
  - [Documents](#documents)

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

## My ToDO list

- [X] CPU
- [X] PPU
- [X] APU
- [ ] NSF Player (cmd/toynes-nsf)
  - like VirtuaNES
- [ ] 6502 compiler
  - like cc65
- [ ] disassembler
- [ ] Interpreter (cmd/toynes-interpreter)
- [ ] sprite extractor (cmd/toynes-sprites)
- [ ] ROM info CLI (cmd/toynes-rominfo)
- [ ] Debugger (like [Mesen's Debugging tools](https://www.mesen.ca/docs/debugging.html))
- [ ] test
  - [ ] [nes-test-roms](https://github.com/christopherpow/nes-test-roms/)
    - like [tetanes README.md](https://github.com/lukexor/tetanes)
  - [ ] go testing (like integration test)
    - [ ] CPU
    - [ ] PPU

## Build

- Install Library
  - portaudio

MacOSX

```shell
brew install portaudio
```

- build

```shell
go build cmd/toynes/main.go
```

## Dependencies

- Dear ImGUI (imgui-go)
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

## Reference

### Emulator

- [libretro/Mesen](https://github.com/libretro/Mesen/)
- [eteran/pretendo](https://github.com/eteran/pretendo)
- [lukexor/tetanes](https://github.com/lukexor/tetanes)
- [ivysnow/virtuanes](https://github.com/ivysnow/virtuanes/)
- [sairoutine/faithjs](https://github.com/sairoutine/faithjs/)

### Documents

- [Nesdev Wiki](https://www.nesdev.org/wiki/Nesdev_Wiki)
- [6502 Instruction Set](https://www.masswerk.at/6502/6502_instruction_set.html)
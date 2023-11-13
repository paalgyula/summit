# 🏔 Summit
### World of Warcraft _3.3.5a_ server emulator - Written in purely GO

"The climbers reached the **summit** of the mountain *after a long and challenging journey*."

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/paalgyula/summit)
![go workflow](https://github.com/paalgyula/summit/actions/workflows/go.yml/badge.svg)
![GitHub top language](https://img.shields.io/github/languages/top/paalgyula/summit)
![Lines of code](https://img.shields.io/tokei/lines/github/paalgyula/summit?style=flat)
![GitHub](https://img.shields.io/github/license/paalgyula/summit)
![GitHub last commit](https://img.shields.io/github/last-commit/paalgyula/summit)


## Modules:

- Authentication/Realmlist server [[**summit-auth**](docs/authserver.md)]
- World Server [summit-world]
- WoW Database converter [datagen]
- Proxy (actually a worm) [[**serworm**](docs/serworm.md)]
- Packet dumper

### Only for fun/education purposes

This project is just a tiny fun project, my free-time fun with GO, Ghidra. I really love this programming language and I've decided to rewrite my abandoned project what I wrote ~15years ago in C++ (that was the original summit emulator for burning crusade) later became [Ascent](https://github.com/SkyFire/ascent_classic) -> ArcEmu ☠ -> [AscEmu](https://github.com/AscEmu/AscEmu)

This project will be pure fun, writing the emulator from scratch after +15years experience 😈 Goal: to just run the wow emulator whenever you wanna play/continue to play or you just want to experiment some low level stuff in go. 

## How to run/develop
The project contains a Makefile which is parameterized to build the project with go 1.21+, the binaries will be placed in `bin/` folder. Later I'm planning to create a **goreleaser** pipeline for github actions to provide some instant binaries too.

The DBC stuff? Hmm... I have an idea to load the dbc in a different way than before. If you check the package: 


### Community

This project is a one man band (because when i'm writing these lines the project is just 3days old). I have an architecture in my head how this tiny project will change the 🗺 and I'll document it soon to here, but feel free to fork this repository and feel free to have fun. 

I've pulled some existing parts from emulators:
- [Azeroth Core](https://github.com/azerothcore/azerothcore-wotlk) - Opcodes
- [TrinityCore](https://github.com/TrinityCore/TrinityCore/tree/3.3.5) - Enums, for DBCs

Thanks for these communities for the research! 🙏


### Why Wotlk?

Because I'm perv a bit. I left the WoW community with this version, so I've decided to jump back in time, and as a linux lover: have a lot of fun 🐧

## Plans/Ideas

- easy to implement/pluggable packet(handler) system
- Some scripting interface (js maybe) to script the dungeons
- exportable metrics
- clustering *(with grid based world)*
- administation interface with gRPC connector hosted on https://buf.build *(planned)*
- federated auth server (one authentication server, anyone can join with a `custom` server)
- Binary file based database no 3rd party sql needed *(plan)*

If you have any question, feel free to contact me:

paalgyula@pm.me | gophers.slack.com/#wow | https://fb.me/nev3rkn0wn | ig: @nev3rkn0wn

# PR-s are welcome!

Made with ♥ by @paalgyula

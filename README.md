# Summit
### World of Warcraft 3.3.5a game server emulator - Written in GO

## Modules:

- Authentication server
- World Server
- WoW Database converter
- Proxy (actually a worm)
- Packet dumper

### Only for fun/education purposes

This project is just a tiny fun project, my free-time fun with GO, Ghidra. I really love this programming language and I've decided to rewrite my abandoned project what I wrote ~15years ago in C++ (that was the original summit emulator for burning crusade) later became ascent -> arcemu -> ☠

Now it shouldn't be dead 🤗 This will be pure fun, to just run the wow emulator whenever you wanna play/continue to play or you just want to experiment some low level stuff. 

## How to run/develop
The project contains a Makefile which is parameterized to build the project with go 1.20+, the binaries will be placed in `bin/` folder. Later I'm planning to create a **goreleaser** pipeline for github actions to provide some instant binaries too.

### Community

This project is a one man band (because when i'm writing these lines the project is just 3days old). I have an architecture in my head how this tiny project will change the 🗺 and I'll document it soon to here, but feel free to fork this repository and feel free to have fun. 

I'm stealing some existing parts from emulators:
- Azeroth Core
- OregonCore
- TrinityCore 
mainly only the packet structure.

### Why Wotlk?

Because I'm perv a bit. I left the WoW community with this version, so I've decided to jump back in time, and as a linux lover: have a lot of fun 🐧

## Plans/Ideas

- easy to implement/pluggable packet(handler) system
- Some scripting interface (js maybe) to script the dungeons
- exportable metrics
- clustering
- administation interface with gRPC connector
- federated auth server (one authentication server, anyone can join with a `custom` server)
- Kubernetes ready scalable world
- Binary file based database no 3rd party sql needed `(WIP)`

If you have any question, feel free to contact me:

paalgyula@pm.me | gophers.slack.com/#wow | fb.me/

# PR-s are welcome!

Made with ♥ by @paalgyula

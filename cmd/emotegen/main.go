// Command emotegen generates the emote tables of the server and the web client from
// Emotes.dbc, EmotesText.dbc, EmotesTextData.dbc and AnimationData.dbc:
//
//	go run ./cmd/emotegen <dbc dir> pkg/summit/world/emotes.gen.go client/src/net/emotes.gen.ts
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/paalgyula/summit/pkg/converter/dbc"
)

func open(dir, name string) *dbc.File {
	f, err := os.Open(dir + "/" + name + ".dbc")
	if err != nil {
		panic(err)
	}
	d, err := dbc.Read(f)
	if err != nil {
		panic(err)
	}
	return d
}

type emote struct {
	id, anim, typ uint32
	name          string
}
type textEmote struct {
	id, emote uint32
	name      string
	texts     [5]string // other→target, other→you, you→target, other, you
}

func main() {
	dir, goOut, tsOut := os.Args[1], os.Args[2], os.Args[3]
	anim := open(dir, "AnimationData")
	animNames := map[uint32]string{}
	for r := 0; r < anim.Records; r++ {
		animNames[anim.Uint32(r, 0)] = anim.String(r, 1)
	}
	em := open(dir, "Emotes")
	var emotes []emote
	for r := 0; r < em.Records; r++ {
		emotes = append(emotes, emote{em.Uint32(r, 0), em.Uint32(r, 2), em.Uint32(r, 4), em.String(r, 1)})
	}
	td := open(dir, "EmotesTextData")
	texts := map[uint32]string{}
	for r := 0; r < td.Records; r++ {
		texts[td.Uint32(r, 0)] = td.String(r, 1)
	}
	et := open(dir, "EmotesText")
	var tes []textEmote
	for r := 0; r < et.Records; r++ {
		t := textEmote{id: et.Uint32(r, 0), name: et.String(r, 1), emote: et.Uint32(r, 2)}
		for i, col := range []int{3, 4, 5, 7, 9} {
			t.texts[i] = texts[et.Uint32(r, col)]
		}
		tes = append(tes, t)
	}
	sort.Slice(tes, func(i, j int) bool { return tes[i].id < tes[j].id })

	// ---- Go
	var g strings.Builder
	g.WriteString("// Code generated from Emotes.dbc / EmotesText.dbc (3.3.5a); DO NOT EDIT.\n\npackage world\n\n")
	g.WriteString("// EmoteType is the Emotes.dbc EmoteType column: 0 one-shot, 1 stand state, 2 looping state.\ntype EmoteType uint32\n\nconst (\n\tEmoteTypeOneShot EmoteType = 0\n\tEmoteTypeStandState EmoteType = 1\n\tEmoteTypeState EmoteType = 2\n)\n\n")
	g.WriteString("// EmoteInfo is one Emotes.dbc row.\ntype EmoteInfo struct {\n\tName string\n\tAnim uint32\n\tType EmoteType\n}\n\n")
	g.WriteString("// Emotes maps an emote id (SMSG_EMOTE payload) to its animation and kind.\n//\n//nolint:gochecknoglobals\nvar Emotes = map[uint32]EmoteInfo{\n")
	for _, e := range emotes {
		fmt.Fprintf(&g, "\t%d: {Name: %q, Anim: %d, Type: %d},\n", e.id, e.name, e.anim, e.typ)
	}
	g.WriteString("}\n\n// TextEmotes maps a text emote id (CMSG_TEXT_EMOTE) to the emote it plays, 0 for text only.\n//\n//nolint:gochecknoglobals\nvar TextEmotes = map[uint32]uint32{\n")
	for _, t := range tes {
		fmt.Fprintf(&g, "\t%d: %d, // %s\n", t.id, t.emote, t.name)
	}
	g.WriteString("}\n")
	if err := os.WriteFile(goOut, []byte(g.String()), 0o644); err != nil {
		panic(err)
	}

	// ---- TS
	var ts strings.Builder
	ts.WriteString("// Generated from Emotes.dbc / EmotesText.dbc / EmotesTextData.dbc / AnimationData.dbc (3.3.5a); do not edit.\n\n")
	ts.WriteString("/** Emotes.dbc: EmoteType 0 one-shot, 1 stand state (sit / sleep / kneel), 2 looping state. */\nexport interface EmoteInfo { name: string; anim: number; type: 0 | 1 | 2 }\n\n")
	ts.WriteString("export const EMOTES: Record<number, EmoteInfo> = {\n")
	for _, e := range emotes {
		fmt.Fprintf(&ts, "  %d: { name: %q, anim: %d, type: %d },\n", e.id, e.name, e.anim, e.typ)
	}
	ts.WriteString("};\n\n/** AnimationData.dbc names of the sequences emotes and stand states play (clip names in the converted models). */\nexport const ANIM_NAMES: Record<number, string> = {\n")
	used := map[uint32]bool{97: true, 100: true, 115: true}
	for _, e := range emotes {
		if e.anim != 0 {
			used[e.anim] = true
		}
	}
	var ids []int
	for id := range used {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	for _, id := range ids {
		fmt.Fprintf(&ts, "  %d: %q,\n", id, animNames[uint32(id)])
	}
	ts.WriteString("};\n\n/** EmotesText.dbc: a slash command, the emote it plays and its chat lines. */\nexport interface TextEmote {\n  id: number;\n  name: string;\n  emote: number;\n  /** [other → target, other → you, you → target, other alone, you alone]; %s are sender then target. */\n  texts: [string, string, string, string, string];\n}\n\nexport const TEXT_EMOTES: Record<number, TextEmote> = {\n")
	for _, t := range tes {
		fmt.Fprintf(&ts, "  %d: { id: %d, name: %q, emote: %d, texts: [%q, %q, %q, %q, %q] },\n", t.id, t.id, t.name, t.emote, t.texts[0], t.texts[1], t.texts[2], t.texts[3], t.texts[4])
	}
	ts.WriteString("};\n")
	if err := os.WriteFile(tsOut, []byte(ts.String()), 0o644); err != nil {
		panic(err)
	}
}

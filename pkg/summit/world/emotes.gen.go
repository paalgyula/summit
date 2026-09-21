// Code generated from Emotes.dbc / EmotesText.dbc (3.3.5a); DO NOT EDIT.

package world

// EmoteType is the Emotes.dbc EmoteType column: 0 one-shot, 1 stand state, 2 looping state.
type EmoteType uint32

const (
	EmoteTypeOneShot    EmoteType = 0
	EmoteTypeStandState EmoteType = 1
	EmoteTypeState      EmoteType = 2
)

// EmoteInfo is one Emotes.dbc row.
type EmoteInfo struct {
	Name string
	Anim uint32
	Type EmoteType
}

// Emotes maps an emote id (SMSG_EMOTE payload) to its animation and kind.
//
//nolint:gochecknoglobals
var Emotes = map[uint32]EmoteInfo{
	0:   {Name: "ONESHOT_NONE", Anim: 0, Type: 0},
	1:   {Name: "ONESHOT_TALK(DNR)", Anim: 60, Type: 0},
	2:   {Name: "ONESHOT_BOW", Anim: 66, Type: 0},
	3:   {Name: "ONESHOT_WAVE(DNR)", Anim: 67, Type: 0},
	4:   {Name: "ONESHOT_CHEER(DNR)", Anim: 68, Type: 0},
	5:   {Name: "ONESHOT_EXCLAMATION(DNR)", Anim: 64, Type: 0},
	6:   {Name: "ONESHOT_QUESTION", Anim: 65, Type: 0},
	7:   {Name: "ONESHOT_EAT", Anim: 61, Type: 0},
	10:  {Name: "STATE_DANCE", Anim: 69, Type: 2},
	11:  {Name: "ONESHOT_LAUGH", Anim: 70, Type: 0},
	12:  {Name: "STATE_SLEEP", Anim: 0, Type: 1},
	13:  {Name: "STATE_SIT", Anim: 0, Type: 1},
	14:  {Name: "ONESHOT_RUDE(DNR)", Anim: 73, Type: 0},
	15:  {Name: "ONESHOT_ROAR(DNR)", Anim: 74, Type: 0},
	16:  {Name: "ONESHOT_KNEEL", Anim: 75, Type: 0},
	17:  {Name: "ONESHOT_KISS", Anim: 76, Type: 0},
	18:  {Name: "ONESHOT_CRY", Anim: 77, Type: 0},
	19:  {Name: "ONESHOT_CHICKEN", Anim: 78, Type: 0},
	20:  {Name: "ONESHOT_BEG", Anim: 79, Type: 0},
	21:  {Name: "ONESHOT_APPLAUD", Anim: 80, Type: 0},
	22:  {Name: "ONESHOT_SHOUT(DNR)", Anim: 81, Type: 0},
	23:  {Name: "ONESHOT_FLEX", Anim: 82, Type: 0},
	24:  {Name: "ONESHOT_SHY(DNR)", Anim: 83, Type: 0},
	25:  {Name: "ONESHOT_POINT(DNR)", Anim: 84, Type: 0},
	26:  {Name: "STATE_STAND", Anim: 0, Type: 1},
	27:  {Name: "STATE_READYUNARMED", Anim: 25, Type: 2},
	28:  {Name: "STATE_WORK_SHEATHED", Anim: 62, Type: 2},
	29:  {Name: "STATE_POINT(DNR)", Anim: 84, Type: 2},
	30:  {Name: "STATE_NONE", Anim: 0, Type: 2},
	33:  {Name: "ONESHOT_WOUND", Anim: 9, Type: 0},
	34:  {Name: "ONESHOT_WOUNDCRITICAL", Anim: 10, Type: 0},
	35:  {Name: "ONESHOT_ATTACKUNARMED", Anim: 16, Type: 0},
	36:  {Name: "ONESHOT_ATTACK1H", Anim: 17, Type: 0},
	37:  {Name: "ONESHOT_ATTACK2HTIGHT", Anim: 18, Type: 0},
	38:  {Name: "ONESHOT_ATTACK2HLOOSE", Anim: 19, Type: 0},
	39:  {Name: "ONESHOT_PARRYUNARMED", Anim: 20, Type: 0},
	43:  {Name: "ONESHOT_PARRYSHIELD", Anim: 24, Type: 0},
	44:  {Name: "ONESHOT_READYUNARMED", Anim: 25, Type: 0},
	45:  {Name: "ONESHOT_READY1H", Anim: 26, Type: 0},
	48:  {Name: "ONESHOT_READYBOW", Anim: 29, Type: 0},
	50:  {Name: "ONESHOT_SPELLPRECAST", Anim: 31, Type: 0},
	51:  {Name: "ONESHOT_SPELLCAST", Anim: 32, Type: 0},
	53:  {Name: "ONESHOT_BATTLEROAR", Anim: 55, Type: 0},
	54:  {Name: "ONESHOT_SPECIALATTACK1H", Anim: 57, Type: 0},
	60:  {Name: "ONESHOT_KICK", Anim: 95, Type: 0},
	61:  {Name: "ONESHOT_ATTACKTHROWN", Anim: 107, Type: 0},
	64:  {Name: "STATE_STUN", Anim: 14, Type: 2},
	65:  {Name: "STATE_DEAD", Anim: 0, Type: 1},
	66:  {Name: "ONESHOT_SALUTE", Anim: 113, Type: 0},
	68:  {Name: "STATE_KNEEL", Anim: 0, Type: 1},
	69:  {Name: "STATE_USESTANDING", Anim: 63, Type: 2},
	70:  {Name: "ONESHOT_WAVE_NOSHEATHE", Anim: 67, Type: 0},
	71:  {Name: "ONESHOT_CHEER_NOSHEATHE", Anim: 68, Type: 0},
	92:  {Name: "ONESHOT_EAT_NOSHEATHE", Anim: 199, Type: 0},
	93:  {Name: "STATE_STUN_NOSHEATHE", Anim: 137, Type: 2},
	94:  {Name: "ONESHOT_DANCE", Anim: 69, Type: 0},
	113: {Name: "ONESHOT_SALUTE_NOSHEATH", Anim: 210, Type: 0},
	133: {Name: "STATE_USESTANDING_NOSHEATHE", Anim: 138, Type: 2},
	153: {Name: "ONESHOT_LAUGH_NOSHEATHE", Anim: 70, Type: 0},
	173: {Name: "STATE_WORK", Anim: 136, Type: 2},
	193: {Name: "STATE_SPELLPRECAST", Anim: 31, Type: 2},
	213: {Name: "ONESHOT_READYRIFLE", Anim: 48, Type: 0},
	214: {Name: "STATE_READYRIFLE", Anim: 48, Type: 2},
	233: {Name: "STATE_WORK_MINING", Anim: 136, Type: 2},
	234: {Name: "STATE_WORK_CHOPWOOD", Anim: 136, Type: 2},
	253: {Name: "STATE_APPLAUD", Anim: 80, Type: 2},
	254: {Name: "ONESHOT_LIFTOFF", Anim: 192, Type: 0},
	273: {Name: "ONESHOT_YES(DNR)", Anim: 185, Type: 0},
	274: {Name: "ONESHOT_NO(DNR)", Anim: 186, Type: 0},
	275: {Name: "ONESHOT_TRAIN(DNR)", Anim: 195, Type: 0},
	293: {Name: "ONESHOT_LAND", Anim: 200, Type: 0},
	313: {Name: "STATE_AT_EASE", Anim: 0, Type: 1},
	333: {Name: "STATE_READY1H", Anim: 26, Type: 2},
	353: {Name: "STATE_SPELLKNEELSTART", Anim: 140, Type: 2},
	373: {Name: "STAND_STATE_SUBMERGED", Anim: 202, Type: 1},
	374: {Name: "ONESHOT_SUBMERGE", Anim: 201, Type: 0},
	375: {Name: "STATE_READY2H", Anim: 27, Type: 2},
	376: {Name: "STATE_READYBOW", Anim: 29, Type: 2},
	377: {Name: "ONESHOT_MOUNTSPECIAL", Anim: 94, Type: 0},
	378: {Name: "STATE_TALK", Anim: 60, Type: 2},
	379: {Name: "STATE_FISHING", Anim: 134, Type: 2},
	380: {Name: "ONESHOT_FISHING", Anim: 133, Type: 0},
	381: {Name: "ONESHOT_LOOT", Anim: 50, Type: 0},
	382: {Name: "STATE_WHIRLWIND", Anim: 126, Type: 2},
	383: {Name: "STATE_DROWNED", Anim: 132, Type: 2},
	384: {Name: "STATE_HOLD_BOW", Anim: 109, Type: 2},
	385: {Name: "STATE_HOLD_RIFLE", Anim: 110, Type: 2},
	386: {Name: "STATE_HOLD_THROWN", Anim: 111, Type: 2},
	387: {Name: "ONESHOT_DROWN", Anim: 131, Type: 0},
	388: {Name: "ONESHOT_STOMP", Anim: 181, Type: 0},
	389: {Name: "ONESHOT_ATTACKOFF", Anim: 87, Type: 0},
	390: {Name: "ONESHOT_ATTACKOFFPIERCE", Anim: 88, Type: 0},
	391: {Name: "STATE_ROAR", Anim: 74, Type: 2},
	392: {Name: "STATE_LAUGH", Anim: 70, Type: 2},
	393: {Name: "ONESHOT_CREATURE_SPECIAL", Anim: 130, Type: 0},
	394: {Name: "ONESHOT_JUMPLANDRUN", Anim: 187, Type: 0},
	395: {Name: "ONESHOT_JUMPEND", Anim: 39, Type: 0},
	396: {Name: "ONESHOT_TALK_NOSHEATHE", Anim: 208, Type: 0},
	397: {Name: "ONESHOT_POINT_NOSHEATHE", Anim: 209, Type: 0},
	398: {Name: "STATE_CANNIBALIZE", Anim: 203, Type: 2},
	399: {Name: "ONESHOT_JUMPSTART", Anim: 37, Type: 0},
	400: {Name: "STATE_DANCESPECIAL", Anim: 211, Type: 2},
	401: {Name: "ONESHOT_DANCESPECIAL", Anim: 211, Type: 0},
	402: {Name: "ONESHOT_CUSTOMSPELL01", Anim: 213, Type: 0},
	403: {Name: "ONESHOT_CUSTOMSPELL02", Anim: 214, Type: 0},
	404: {Name: "ONESHOT_CUSTOMSPELL03", Anim: 215, Type: 0},
	405: {Name: "ONESHOT_CUSTOMSPELL04", Anim: 216, Type: 0},
	406: {Name: "ONESHOT_CUSTOMSPELL05", Anim: 217, Type: 0},
	407: {Name: "ONESHOT_CUSTOMSPELL06", Anim: 218, Type: 0},
	408: {Name: "ONESHOT_CUSTOMSPELL07", Anim: 219, Type: 0},
	409: {Name: "ONESHOT_CUSTOMSPELL08", Anim: 220, Type: 0},
	410: {Name: "ONESHOT_CUSTOMSPELL09", Anim: 221, Type: 0},
	411: {Name: "ONESHOT_CUSTOMSPELL10", Anim: 222, Type: 0},
	412: {Name: "STATE_EXCLAIM", Anim: 64, Type: 2},
	413: {Name: "STATE_DANCE_CUSTOM", Anim: 0, Type: 2},
	415: {Name: "STATE_SIT_CHAIR_MED", Anim: 103, Type: 2},
	416: {Name: "STATE_CUSTOM_SPELL_01", Anim: 213, Type: 2},
	417: {Name: "STATE_CUSTOM_SPELL_02", Anim: 214, Type: 2},
	418: {Name: "STATE_EAT", Anim: 61, Type: 2},
	419: {Name: "STATE_CUSTOM_SPELL_04", Anim: 216, Type: 2},
	420: {Name: "STATE_CUSTOM_SPELL_03", Anim: 215, Type: 2},
	421: {Name: "STATE_CUSTOM_SPELL_05", Anim: 217, Type: 2},
	422: {Name: "STATE_SPELLEFFECT_HOLD", Anim: 158, Type: 2},
	423: {Name: "STATE_EAT_NO_SHEATHE", Anim: 199, Type: 2},
	424: {Name: "STATE_MOUNT", Anim: 91, Type: 1},
	425: {Name: "STATE_READY2HL", Anim: 28, Type: 2},
	426: {Name: "STATE_SIT_CHAIR_HIGH", Anim: 104, Type: 2},
	427: {Name: "STATE_FALL", Anim: 40, Type: 2},
	428: {Name: "STATE_LOOT", Anim: 188, Type: 2},
	429: {Name: "STATE_SUBMERGED", Anim: 202, Type: 2},
	430: {Name: "ONESHOT_COWER(DNR)", Anim: 225, Type: 0},
	431: {Name: "STATE_COWER", Anim: 225, Type: 2},
	432: {Name: "ONESHOT_USESTANDING", Anim: 63, Type: 0},
	433: {Name: "STATE_STEALTH_STAND", Anim: 120, Type: 2},
	434: {Name: "ONESHOT_OMNICAST_GHOUL (W/SOUND", Anim: 54, Type: 0},
	435: {Name: "ONESHOT_ATTACKBOW", Anim: 46, Type: 0},
	436: {Name: "ONESHOT_ATTACKRIFLE", Anim: 49, Type: 0},
	437: {Name: "STATE_SWIM_IDLE", Anim: 41, Type: 2},
	438: {Name: "STATE_ATTACK_UNARMED", Anim: 16, Type: 2},
	439: {Name: "ONESHOT_SPELLCAST (W/SOUND)", Anim: 32, Type: 0},
	440: {Name: "ONESHOT_DODGE", Anim: 30, Type: 0},
	441: {Name: "ONESHOT_PARRY1H", Anim: 21, Type: 0},
	442: {Name: "ONESHOT_PARRY2H", Anim: 22, Type: 0},
	443: {Name: "ONESHOT_PARRY2HL", Anim: 28, Type: 0},
	444: {Name: "STATE_FLYFALL", Anim: 269, Type: 2},
	445: {Name: "ONESHOT_FLYDEATH", Anim: 230, Type: 0},
	446: {Name: "STATE_FLY_FALL", Anim: 269, Type: 2},
	447: {Name: "ONESHOT_FLY_SIT_GROUND_DOWN", Anim: 325, Type: 0},
	448: {Name: "ONESHOT_FLY_SIT_GROUND_UP", Anim: 327, Type: 0},
	449: {Name: "ONESHOT_EMERGE", Anim: 224, Type: 0},
	450: {Name: "ONESHOT_DRAGONSPIT", Anim: 182, Type: 0},
	451: {Name: "STATE_SPECIALUNARMED", Anim: 118, Type: 2},
	452: {Name: "ONESHOT_FLYGRAB", Anim: 455, Type: 0},
	453: {Name: "STATE_FLYGRABCLOSED", Anim: 456, Type: 2},
	454: {Name: "ONESHOT_FLYGRABTHROWN", Anim: 457, Type: 0},
	455: {Name: "STATE_FLY_SIT_GROUND", Anim: 326, Type: 2},
	456: {Name: "STATE_WALKBACKWARDS", Anim: 13, Type: 2},
	457: {Name: "ONESHOT_FLYTALK", Anim: 289, Type: 0},
	458: {Name: "ONESHOT_FLYATTACK1H", Anim: 246, Type: 0},
	459: {Name: "STATE_CUSTOMSPELL08", Anim: 220, Type: 2},
	460: {Name: "ONESHOT_FLY_DRAGONSPIT", Anim: 411, Type: 0},
	461: {Name: "STATE_SIT_CHAIR_LOW", Anim: 102, Type: 2},
	462: {Name: "ONE_SHOT_STUN", Anim: 14, Type: 0},
	463: {Name: "ONESHOT_SPELLCAST_OMNI", Anim: 54, Type: 0},
	465: {Name: "STATE_READYTHROWN", Anim: 108, Type: 2},
	466: {Name: "ONESHOT_WORK_CHOPWOOD", Anim: 62, Type: 0},
	467: {Name: "ONESHOT_WORK_MINING", Anim: 62, Type: 0},
	468: {Name: "STATE_SPELL_CHANNEL_OMNI", Anim: 125, Type: 2},
	469: {Name: "STATE_SPELL_CHANNEL_DIRECTED", Anim: 124, Type: 2},
	470: {Name: "STAND_STATE_NONE", Anim: 0, Type: 1},
	471: {Name: "STATE_READYJOUST", Anim: 476, Type: 1},
	473: {Name: "STATE_STRANGULATE", Anim: 474, Type: 2},
	474: {Name: "STATE_READYSPELLOMNI", Anim: 52, Type: 2},
	475: {Name: "STATE_HOLD_JOUST", Anim: 478, Type: 2},
	476: {Name: "ONESHOT_CRY (JAINA PROUDMOORE ONLY)", Anim: 77, Type: 0},
}

// TextEmotes maps a text emote id (CMSG_TEXT_EMOTE) to the emote it plays, 0 for text only.
//
//nolint:gochecknoglobals
var TextEmotes = map[uint32]uint32{
	1:   0,   // AGREE
	2:   0,   // AMAZE
	3:   14,  // ANGRY
	4:   0,   // APOLOGIZE
	5:   21,  // APPLAUD
	6:   24,  // BASHFUL
	7:   0,   // BECKON
	8:   20,  // BEG
	9:   0,   // BITE
	10:  0,   // BLEED
	11:  0,   // BLINK
	12:  24,  // BLUSH
	13:  0,   // BONK
	14:  0,   // BORED
	15:  0,   // BOUNCE
	16:  0,   // BRB
	17:  2,   // BOW
	18:  0,   // BURP
	19:  3,   // BYE
	20:  11,  // CACKLE
	21:  4,   // CHEER
	22:  19,  // CHICKEN
	23:  11,  // CHUCKLE
	24:  21,  // CLAP
	25:  6,   // CONFUSED
	26:  5,   // CONGRATULATE
	27:  0,   // COUGH
	28:  431, // COWER
	29:  0,   // CRACK
	30:  0,   // CRINGE
	31:  18,  // CRY
	32:  6,   // CURIOUS
	33:  2,   // CURTSEY
	34:  10,  // DANCE
	35:  7,   // DRINK
	36:  0,   // DROOL
	37:  7,   // EAT
	38:  0,   // EYE
	39:  0,   // FART
	40:  0,   // FIDGET
	41:  23,  // FLEX
	42:  0,   // FROWN
	43:  5,   // GASP
	44:  0,   // GAZE
	45:  11,  // GIGGLE
	46:  0,   // GLARE
	47:  11,  // GLOAT
	48:  3,   // GREET
	49:  0,   // GRIN
	50:  0,   // GROAN
	51:  20,  // GROVEL
	52:  11,  // GUFFAW
	53:  3,   // HAIL
	54:  0,   // HAPPY
	55:  3,   // HELLO
	56:  0,   // HUG
	57:  0,   // HUNGRY
	58:  17,  // KISS
	59:  68,  // KNEEL
	60:  11,  // LAUGH
	61:  12,  // LAYDOWN
	62:  0,   // MASSAGE
	63:  0,   // MOAN
	64:  0,   // MOON
	65:  18,  // MOURN
	66:  274, // NO
	67:  273, // NOD
	68:  0,   // NOSEPICK
	69:  0,   // PANIC
	70:  0,   // PEER
	71:  20,  // PLEAD
	72:  25,  // POINT
	73:  0,   // POKE
	74:  16,  // PRAY
	75:  15,  // ROAR
	76:  11,  // ROFL
	77:  14,  // RUDE
	78:  66,  // SALUTE
	79:  0,   // SCRATCH
	80:  0,   // SEXY
	81:  0,   // SHAKE
	82:  22,  // SHOUT
	83:  6,   // SHRUG
	84:  24,  // SHY
	85:  0,   // SIGH
	86:  13,  // SIT
	87:  12,  // SLEEP
	88:  0,   // SNARL
	89:  0,   // SPIT
	90:  0,   // STARE
	91:  0,   // SURPRISED
	92:  20,  // SURRENDER
	93:  1,   // TALK
	94:  5,   // TALKEX
	95:  6,   // TALKQ
	96:  0,   // TAP
	97:  1,   // THANK
	98:  0,   // THREATEN
	99:  0,   // TIRED
	100: 4,   // VICTORY
	101: 3,   // WAVE
	102: 3,   // WELCOME
	103: 0,   // WHINE
	104: 0,   // WHISTLE
	105: 0,   // WORK
	106: 0,   // YAWN
	107: 6,   // BOGGLE
	108: 0,   // CALM
	109: 0,   // COLD
	110: 0,   // COMFORT
	111: 0,   // CUDDLE
	112: 0,   // DUCK
	113: 14,  // INSULT
	114: 0,   // INTRODUCE
	115: 0,   // JK
	116: 0,   // LICK
	117: 0,   // LISTEN
	118: 6,   // LOST
	119: 0,   // MOCK
	120: 6,   // PONDER
	121: 0,   // POUNCE
	122: 0,   // PRAISE
	123: 0,   // PURR
	124: 6,   // PUZZLE
	125: 0,   // RAISE
	126: 0,   // READY
	127: 0,   // SHIMMY
	128: 0,   // SHIVER
	129: 0,   // SHOO
	130: 0,   // SLAP
	131: 0,   // SMIRK
	132: 0,   // SNIFF
	133: 0,   // SNUB
	134: 0,   // SOOTHE
	135: 0,   // STINK
	136: 19,  // TAUNT
	137: 0,   // TEASE
	138: 0,   // THIRSTY
	139: 0,   // VETO
	140: 0,   // SNICKER
	141: 26,  // STAND
	142: 0,   // TICKLE
	143: 18,  // VIOLIN
	163: 0,   // SMILE
	183: 14,  // RASP
	203: 0,   // PITY
	204: 15,  // GROWL
	205: 0,   // BARK
	223: 430, // SCARED
	224: 0,   // FLOP
	225: 0,   // LOVE
	226: 0,   // MOO
	243: 21,  // COMMEND
	264: 275, // TRAIN
	303: 22,  // HELPME
	304: 25,  // INCOMING
	305: 25,  // CHARGE
	306: 22,  // FLEE
	307: 15,  // ATTACKMYTARGET
	323: 1,   // OOM
	324: 1,   // FOLLOW
	325: 1,   // WAIT
	326: 1,   // HEALME
	327: 25,  // OPENFIRE
	328: 24,  // FLIRT
	329: 1,   // JOKE
	343: 21,  // GOLFCLAP
	363: 0,   // WINK
	364: 0,   // PAT
	365: 0,   // SERIOUS
	366: 377, // MOUNTSPECIAL
	367: 0,   // GOODLUCK
	368: 25,  // BLAME
	369: 0,   // BLANK
	370: 0,   // BRANDISH
	371: 0,   // BREATH
	372: 274, // DISAGREE
	373: 274, // DOUBT
	374: 0,   // EMBARRASS
	375: 0,   // ENCOURAGE
	376: 0,   // ENEMY
	377: 0,   // EYEBROW
	378: 7,   // TOAST
	379: 18,  // FAIL
	380: 0,   // HIGHFIVE
	381: 0,   // ABSENT
	382: 0,   // ARM
	383: 0,   // AWE
	384: 0,   // BACKPACK
	385: 0,   // BADFEELING
	386: 0,   // CHALLENGE
	387: 0,   // CHUG
	389: 0,   // DING
	390: 0,   // FACEPALM
	391: 0,   // FAINT
	392: 0,   // GO
	393: 0,   // GOING
	394: 0,   // GLOWER
	395: 0,   // HEADACHE
	396: 0,   // HICCUP
	398: 0,   // HISS
	399: 0,   // HOLDHAND
	401: 0,   // HURRY
	402: 0,   // IDEA
	403: 0,   // JEALOUS
	404: 0,   // LUCK
	405: 0,   // MAP
	406: 20,  // MERCY
	407: 0,   // MUTTER
	408: 0,   // NERVOUS
	409: 0,   // OFFER
	410: 0,   // PET
	411: 0,   // PINCH
	413: 0,   // PROUD
	414: 0,   // PROMISE
	415: 0,   // PULSE
	416: 0,   // PUNCH
	417: 0,   // POUT
	418: 0,   // REGRET
	420: 0,   // REVENGE
	421: 0,   // ROLLEYES
	422: 0,   // RUFFLE
	423: 0,   // SAD
	424: 0,   // SCOFF
	425: 0,   // SCOLD
	426: 0,   // SCOWL
	427: 0,   // SEARCH
	428: 0,   // SHAKEFIST
	429: 0,   // SHIFTY
	430: 0,   // SHUDDER
	431: 0,   // SIGNAL
	432: 0,   // SILENCE
	433: 1,   // SING
	434: 0,   // SMACK
	435: 0,   // SNEAK
	436: 0,   // SNEEZE
	437: 0,   // SNORT
	438: 0,   // SQUEAL
	439: 0,   // STOPATTACK
	440: 0,   // SUSPICIOUS
	441: 0,   // THINK
	442: 0,   // TRUCE
	443: 0,   // TWIDDLE
	444: 0,   // WARN
	445: 0,   // SNAP
	446: 0,   // CHARM
	447: 0,   // COVEREARS
	448: 0,   // CROSSARMS
	449: 0,   // LOOK
	450: 25,  // OBJECT
	451: 0,   // SWEAT
	453: 1,   // YW
}

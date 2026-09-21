package world

import (
	"fmt"

	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog/log"
)

type (
	handlePacket  = func(wow.PacketData)
	handlePacket2 = func(*wow.Packet)
	handleCommand = func()
)

// ExternalPacketFunc register packet for external processing.
type ExternalPacketFunc = func(*WorldSession, wow.OpCode, []byte)

type PacketHandler struct {
	Opcode  wow.OpCode
	Handler any
}

func (gc *WorldSession) RegisterHandlers(handlers ...PacketHandler) {
	if len(handlers) < int(wow.NumMsgTypes) {
		origSize := len(handlers)

		additional := int(wow.NumMsgTypes) - len(handlers)
		for i := 0; i < additional; i++ {
			handlers = append(handlers, PacketHandler{
				Opcode:  wow.OpCode(origSize + i + 1),
				Handler: "none",
			})
		}
	}

	for _, oh := range handlers {
		if len(gc.opcodes) <= int(oh.Opcode) {
			log.Printf("opcode table too short: 0x%03x", oh.Opcode)

			continue
		}

		gc.opcodes.Handle(oh.Opcode, oh.Handler)
	}

	gc.opcodes.Handle(wow.ClientPing, gc.PingHandler)
	gc.opcodes.Handle(wow.ClientAuthSession, gc.AuthSessionHandler)
	gc.opcodes.Handle(wow.ClientCharEnum, gc.SendCharacterEnum)
	gc.opcodes.Handle(wow.ClientCharCreate, gc.CreateCharacter)
	gc.opcodes.Handle(wow.ClientRealmSplit, gc.HandleRealmSplit)
	gc.opcodes.Handle(wow.ClientPlayerLogin, gc.HandlePlayerLogin)

	// Logout handler
	gc.opcodes.Handle(wow.ClientLogoutRequest, gc.HandleLogoutRequest)

	// Chat handler
	gc.opcodes.Handle(wow.ClientMessagechat, gc.HandleMessageChat)

	// Channel handlers
	gc.opcodes.Handle(wow.ClientJoinChannel, gc.HandleJoinChannel)
	gc.opcodes.Handle(wow.ClientLeaveChannel, gc.HandleLeaveChannel)
	gc.opcodes.Handle(wow.ClientChannelList, gc.HandleChannelList)
	gc.opcodes.Handle(wow.ClientChannelPassword, gc.HandleChannelPassword)
	gc.opcodes.Handle(wow.ClientChannelSetOwner, gc.HandleChannelSetOwner)
	gc.opcodes.Handle(wow.ClientChannelOwner, gc.HandleChannelOwner)
	gc.opcodes.Handle(wow.ClientChannelModerator, gc.HandleChannelModerator)
	gc.opcodes.Handle(wow.ClientChannelUnmoderator, gc.HandleChannelUnmoderator)
	gc.opcodes.Handle(wow.ClientChannelMute, gc.HandleChannelMute)
	gc.opcodes.Handle(wow.ClientChannelUnmute, gc.HandleChannelUnmute)
	gc.opcodes.Handle(wow.ClientChannelKick, gc.HandleChannelKick)
	gc.opcodes.Handle(wow.ClientChannelBan, gc.HandleChannelBan)
	gc.opcodes.Handle(wow.ClientChannelUnban, gc.HandleChannelUnban)

	// LFG handlers
	gc.opcodes.Handle(wow.ClientLfgJoin, gc.HandleLfgJoin)
	gc.opcodes.Handle(wow.ClientLfgLeave, gc.HandleLfgLeave)
	gc.opcodes.Handle(wow.ClientLfgSetRoles, gc.HandleLfgSetRoles)
	gc.opcodes.Handle(wow.ClientLfgGetStatus, gc.HandleLfgGetStatus)

	// Name query
	gc.opcodes.Handle(wow.ClientNameQuery, gc.HandleNameQuery)
	gc.opcodes.Handle(wow.ClientCreatureQuery, gc.HandleCreatureQuery)

	// Targeting
	gc.opcodes.Handle(wow.ClientSetSelection, gc.HandleSetSelection)

	// Action bar
	gc.opcodes.Handle(wow.ClientSetActionButton, gc.HandleSetActionButton)

	// Emotes
	gc.opcodes.Handle(wow.ClientTextEmote, gc.HandleTextEmote)
	gc.opcodes.Handle(wow.ClientEmote, gc.HandleEmote)

	// Combat handlers
	gc.opcodes.Handle(wow.ClientAttackswing, gc.HandleAttackSwing)
	gc.opcodes.Handle(wow.ClientAttackstop, gc.HandleAttackStop)

	// Spell handler
	gc.opcodes.Handle(wow.ClientCastSpell, gc.HandleCastSpell)

	// Item handlers
	gc.opcodes.Handle(wow.ClientAutoequipItem, gc.HandleAutoEquipItem)
	gc.opcodes.Handle(wow.ClientAutoequipItemSlot, gc.HandleAutoEquipItemSlot)
	gc.opcodes.Handle(wow.ClientSwapInvItem, gc.HandleSwapInvItem)
	gc.opcodes.Handle(wow.ClientSwapItem, gc.HandleSwapItem)
	gc.opcodes.Handle(wow.ClientDestroyitem, gc.HandleDestroyItem)
	gc.opcodes.Handle(wow.ClientItemQuerySingle, gc.HandleItemQuerySingle)

	// Teleport/area trigger handlers
	gc.opcodes.Handle(wow.ClientAreatrigger, gc.HandleAreaTriggerOpcode)
	gc.opcodes.Handle(wow.MsgMoveWorldportAck, gc.HandleMoveWorldportAck)
	gc.opcodes.Handle(wow.MsgMoveTeleportAck, gc.HandleTeleportAck)

	// Movement handlers - all use the same handler function
	movementOpcodes := []wow.OpCode{
		wow.MsgMoveStartForward,
		wow.MsgMoveStartBackward,
		wow.MsgMoveStop,
		wow.MsgMoveStartStrafeLeft,
		wow.MsgMoveStartStrafeRight,
		wow.MsgMoveStopStrafe,
		wow.MsgMoveStartTurnLeft,
		wow.MsgMoveStartTurnRight,
		wow.MsgMoveStopTurn,
		wow.MsgMoveStartPitchUp,
		wow.MsgMoveStartPitchDown,
		wow.MsgMoveStopPitch,
		wow.MsgMoveFallLand,
		wow.MsgMoveJump,
		wow.MsgMoveSetFacing,
		wow.MsgMoveSetPitch,
		wow.MsgMoveStartSwim,
		wow.MsgMoveStopSwim,
		wow.MsgMoveHeartbeat,
		wow.MsgMoveStartAscend,
		wow.MsgMoveStopAscend,
		wow.MsgMoveStartDescend,
	}

	for _, op := range movementOpcodes {
		gc.opcodes.Handle(op, handlePacket(func(data wow.PacketData) {
			gc.HandleMovementOpcodes(op, data)
		}))
	}

	// Speed change handlers
	speedOpcodes := []wow.OpCode{
		wow.MsgMoveSetRunSpeed,
		wow.MsgMoveSetRunBackSpeed,
		wow.MsgMoveSetWalkSpeed,
		wow.MsgMoveSetSwimSpeed,
		wow.MsgMoveSetSwimBackSpeed,
		wow.MsgMoveSetTurnRate,
		wow.MsgMoveSetFlightSpeed,
		wow.MsgMoveSetFlightBackSpeed,
	}

	for _, op := range speedOpcodes {
		gc.opcodes.Handle(op, handlePacket(func(data wow.PacketData) {
			gc.HandleMovementSpeed(op, data)
		}))
	}

	// Special movement handlers
	gc.opcodes.Handle(wow.ClientMoveFallReset, handlePacket(func(data wow.PacketData) {
		gc.HandleMovementOpcodes(wow.ClientMoveFallReset, data)
	}))
	gc.opcodes.Handle(wow.ClientMoveTimeSkipped, gc.HandleMoveTimeSkipped)

	// Quest handlers
	gc.opcodes.Handle(wow.ClientQuestgiverHello, gc.HandleQuestgiverHello)
	gc.opcodes.Handle(wow.ClientQuestgiverQueryQuest, gc.HandleQuestgiverQueryQuest)
	gc.opcodes.Handle(wow.ClientQuestgiverAcceptQuest, gc.HandleQuestgiverAcceptQuest)
	gc.opcodes.Handle(wow.ClientQuestgiverCompleteQuest, gc.HandleQuestgiverCompleteQuest)
	gc.opcodes.Handle(wow.ClientQuestgiverRequestReward, gc.HandleQuestgiverRequestReward)
	gc.opcodes.Handle(wow.ClientQuestgiverChooseReward, gc.HandleQuestgiverChooseReward)
	gc.opcodes.Handle(wow.ClientQuestgiverCancel, gc.HandleQuestgiverCancel)
	gc.opcodes.Handle(wow.ClientQuestgiverStatusQuery, gc.HandleQuestgiverStatusQuery)
	gc.opcodes.Handle(wow.ClientQuestlogRemoveQuest, gc.HandleQuestLogRemoveQuest)
}

func (gc *WorldSession) Handle(pkt *wow.Packet) {
	wow.GetPacketDumper().Write(pkt.Opcode(), pkt.Bytes())

	handle := gc.opcodes.Get(pkt.Opcode())
	if handle == nil {
		// return errors.New("no handler record found")
		gc.log.Warn().Msgf("no handler record found: 0x%04x", pkt.OpCode())

		return
	}

	switch t := handle.Handler.(type) {
	case string:
		gc.log.Warn().
			Str("packet", pkt.Opcode().String()).
			Str("handler", t).
			Msg("handler defined as string")
	case handlePacket:
		t(pkt.Bytes())
	case handlePacket2: // func(*wow.Packet)
		t(pkt)
	case handleCommand:
		t()
	case ExternalPacketFunc:
		t(gc, pkt.Opcode(), pkt.Bytes())
	default:
		gc.log.Error().Msgf("handler function is not defined: %s", t)
		gc.log.Error().
			Type("packet", pkt.Opcode().String()).
			Str("opcode", fmt.Sprintf("0x%04x", pkt.OpCode())).
			Str("handler", fmt.Sprintf("%+v %T", handle, handle.Handler)).
			Msgf("handler type not handled")
	}

	// switch oc {
	// case ClientPing:
	// 	gc.PingHandler()
	// case ClientAuthSession:
	// 	gc.AuthSessionHandler(data)
	// case ClientCharEnum:
	// 	gc.ListCharacters()
	// default:

	// }
}

package world

import (
	"fmt"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

type Updater struct {
	UpdateData  []any
	updateFlags uint8
}

func (upd *Updater) buildMovementUpdate(unit any, pkt *wow.Packet) {
	var o *object.Object
	var u *object.Unit
	var p *player.Player

	switch t := unit.(type) {
	case *player.Player:
		o = t.Object
		p = t
	case *object.Object:
		o = t
	default:
		panic(fmt.Sprintf("unknown type: %T", unit))
	}

	pkt.Write(wow.UpdateTypeMovement)
	pkt.Write(o.Guid())

	moveFlags := wow.MovementFlagNone

	pkt.Write(upd.updateFlags) // update flags

	if upd.updateFlags&wow.UpdateFlagLiving != 0 {
		switch o.Guid().TypeID() {
		case wow.TypeIDUnit:
			{
				moveFlags = o.MovementFlags()
				moveFlags &= ^wow.MovementFlagOnTransport
			}
			break
		case wow.TypeIDPlayer:
			{
				moveFlags = o.MovementFlags()

				if p.Transport() != nil {
					moveFlags |= wow.MovementFlagOnTransport
				} else {
					moveFlags &= ^wow.MovementFlagOnTransport
				}

			}
			break
		}

		pkt.Write(moveFlags)                      // movement flags
		pkt.WriteOne(0)                           // movemoveFlags
		pkt.Write(uint32(time.Now().UnixMilli())) // time (in milliseconds)
	}

	if upd.updateFlags&wow.UpdateFlagHasPosition != 0 {
		if upd.updateFlags&wow.UpdateFlagTransport != 0 &&
			o.GameObjectType() == wow.GameObjectTypeMoTransport {
			pkt.Write(float32(0))
			pkt.Write(float32(0))
			pkt.Write(float32(0))
			// *data << float(((WorldObject*)this)->GetOrientation());
			pkt.Write(float32(0)) // Orientation
		} else {
			// *data << float(((WorldObject*)this)->GetPositionX());
			pkt.Write(float32(0))
			// *data << float(((WorldObject*)this)->GetPositionY());
			pkt.Write(float32(0))
			// *data << float(((WorldObject*)this)->GetPositionZ());
			pkt.Write(float32(0))
			// *data << float(((WorldObject*)this)->GetOrientation());
			pkt.Write(float32(0))
		}
	}

	// // 0x20
	if upd.updateFlags&wow.UpdateFlagLiving != 0 {

		// {
		//     // 0x00000200
		//     if (moveFlags & MOVEMENTFLAG_ONTRANSPORT)
		//     {
		//         if (GetTypeId() == TYPEID_PLAYER)
		//         {
		//             *data << (uint64)ToPlayer()->GetTransport()->GetGUID();
		//             *data << (float)ToPlayer()->GetTransOffsetX();
		//             *data << (float)ToPlayer()->GetTransOffsetY();
		//             *data << (float)ToPlayer()->GetTransOffsetZ();
		//             *data << (float)ToPlayer()->GetTransOffsetO();
		//             *data << (uint32)ToPlayer()->GetTransTime();
		//         }
		//         //Oregon currently not have support for other than player on transport
		//     }

		//     // 0x02200000
		//     if (moveFlags & (MOVEMENTFLAG_SWIMMING | MOVEMENTFLAG_FLYING2))
		//     {
		//         if (GetTypeId() == TYPEID_PLAYER)
		//             *data << (float)ToPlayer()->m_movementInfo.s_pitch;
		//         else
		//             *data << float(0);                          // is't part of movement packet, we must store and send it...
		//     }

		if o.Guid().TypeID() == wow.TypeIDPlayer {
			//         *data << (uint32)ToPlayer()->m_movementInfo.GetFallTime();
			//     else
			//         *data << uint32(0);                             // last fall time
			pkt.Write(uint32(0))

			//     // 0x00001000
			//     if (moveFlags & MOVEMENTFLAG_FALLING)
			//     {
			//         if (GetTypeId() == TYPEID_PLAYER)
			//         {
			//             *data << float(ToPlayer()->m_movementInfo.j_velocity);
			//             *data << float(ToPlayer()->m_movementInfo.j_sinAngle);
			//             *data << float(ToPlayer()->m_movementInfo.j_cosAngle);
			//             *data << float(ToPlayer()->m_movementInfo.j_xyspeed);
			//         }
			//         else
			//         {
			//             *data << float(0);
			//             *data << float(0);
			//             *data << float(0);
			//             *data << float(0);
			//         }
			//     }

			//     // 0x04000000
			//     if (moveFlags & MOVEMENTFLAG_SPLINE_ELEVATION)
			//     {
			//         if (GetTypeId() == TYPEID_PLAYER)
			//             *data << float(ToPlayer()->m_movementInfo.u_unk1);
			//         else
			//             *data << float(0);
			//     }

			//     // Unit speeds
			//     *data << ((Unit*)this)->GetSpeed(MOVE_WALK);
			pkt.Write(u.GetSpeed(wow.MoveTypeWalk))
			//     *data << ((Unit*)this)->GetSpeed(MOVE_RUN);
			pkt.Write(u.GetSpeed(wow.MoveTypeRun))
			//     *data << ((Unit*)this)->GetSpeed(MOVE_RUN_BACK);
			pkt.Write(u.GetSpeed(wow.MoveTypeRunBack))
			//     *data << ((Unit*)this)->GetSpeed(MOVE_SWIM);
			pkt.Write(u.GetSpeed(wow.MoveTypeSwim))
			//     *data << ((Unit*)this)->GetSpeed(MOVE_SWIM_BACK);
			pkt.Write(u.GetSpeed(wow.MoveTypeSwimBack))
			//     *data << ((Unit*)this)->GetSpeed(MOVE_FLIGHT);
			pkt.Write(u.GetSpeed(wow.MoveTypeFlight))
			//     *data << ((Unit*)this)->GetSpeed(MOVE_FLIGHT_BACK);
			pkt.Write(u.GetSpeed(wow.MoveTypeFlightBack))
			//     *data << ((Unit*)this)->GetSpeed(MOVE_TURN_RATE);
			pkt.Write(u.GetSpeed(wow.MoveTypeTurnRate))

			//     // 0x08000000
			//     if (moveFlags & MOVEMENTFLAG_SPLINE_ENABLED)
			//         Movement::PacketBuilder::WriteCreate(*((Unit*)this)->movespline, *data);
		}
	}

	// // 0x8
	if upd.updateFlags&wow.UpdateFlagLowGuid != 0 {
		switch o.Guid().TypeID() {
		case wow.TypeIDObject, wow.TypeIDItem, wow.TypeIDContainer,
			wow.TypeIDGameObject, wow.TypeIDDynamicoObject, wow.TypeIDCorpse:
			pkt.Write(o.Guid().Entry()) // GetGUIDLow()
		case wow.TypeIDUnit:
			// *data << uint32(0x0000000B); // unk, can be 0xB or 0xC
			pkt.WriteUint32(0x0B)
		case wow.TypeIDPlayer:
			if upd.updateFlags&wow.UpdateFlagSelf != 0 {
				// *data << uint32(0x00000015); // unk, can be 0x15 or 0x22
				pkt.WriteUint32(0x15)
			} else {
				// *data << uint32(0x00000008); // unk, can be 0x7 or 0x8
				pkt.WriteUint32(0x8)
			}
		default:
			// *data << uint32(0x00000000); // unk
			pkt.WriteUint32(0x00000000)
		}
	}

	// // 0x10
	if upd.updateFlags&wow.UpdateFlagHighGuid != 0 {
		switch o.Guid().TypeID() {
		case wow.TypeIDObject, wow.TypeIDItem, wow.TypeIDContainer,
			wow.TypeIDGameObject, wow.TypeIDDynamicoObject, wow.TypeIDCorpse:
			pkt.Write(o.Guid().High()) // GetGUIDHigh()
		default:
			pkt.WriteUint32(0x00) // unk
		}
	}

	// // 0x4
	// if (updateFlags & UPDATEFLAG_HAS_ATTACKING_TARGET)  // packed guid (probably target guid)
	// {
	//     if (Unit const* me = ToUnit())
	//     {
	//         if (me->GetVictim())
	//             *data << me->GetVictim()->GetPackGUID();
	//         else
	//             *data << uint8(0);
	//     }
	//     else
	//         *data << uint8(0);
	// }

	// // 0x2
	// if (updateFlags & UPDATEFLAG_TRANSPORT)
	// {
	//     *data << uint32(getMSTime());                       // ms time
	// }
}

func (o *Updater) BuildUpdateObject(player *player.Player) *wow.Packet {
	p := wow.NewPacket(wow.ServerUpdateObject)

	p.WriteUint32(len(o.UpdateData))
	p.WriteOne(0) // Has transport

	// if (!m_outOfRangeGUIDs.empty())
	// {
	//     buf << (uint8) UPDATETYPE_OUT_OF_RANGE_OBJECTS;
	//     buf << (uint32) m_outOfRangeGUIDs.size();

	//     for (std::set<uint64>::const_iterator i = m_outOfRangeGUIDs.begin(); i != m_outOfRangeGUIDs.end(); ++i)
	//     {
	//         // buf << i->WriteAsPacked();
	//         buf << (uint8)0xFF;
	//         buf << *i;
	//     }
	// }

	o.buildMovementUpdate(player, p)

	return p
}

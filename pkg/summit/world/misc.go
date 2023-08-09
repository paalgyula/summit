package world

import (
	bitmask "github.com/paalgyula/summit/pkg/summit/shared/bitmask"
	"github.com/paalgyula/summit/pkg/wow"
)

func (gc *GameClient) PingHandler() {
	gc.SendPayload(int(wow.ServerPong), make([]byte, 2))
}

func (gc *GameClient) SendAccountDataTimes(mask uint32) {
	w := wow.NewPacket(wow.ServerAccountDataTimes)
	// w.Write(unknown)
	w.Write(uint32(0)) // unix game time
	w.Write(uint8(1))
	w.Write(uint32(mask))
	var i uint32
	num_acct_data_types := uint32(wow.NUM_ACCOUNT_DATA_TYPES)
	for i = 0; i < num_acct_data_types; i++ {
		if bitmask.HasOne(mask, bitmask.FlagAt(i)) {
			w.Write(uint32(gc.acc.Metadata[i].Time)) // also unix time
		}
		// do we send blank uint32s to pad the rest??
	}
	gc.SendPayload(int(wow.ServerAccountDataTimes), w.Bytes())
}

func (gc *GameClient) AccountDataTimesHandler() {
	gc.SendAccountDataTimes(wow.GLOBAL_CACHE_MASK)
}

func (gc *GameClient) UpdateAccountDataHandler(recv_data wow.PacketData) {
	var a_type, a_timestamp, decompressed_size uint32
	r := recv_data.Reader()
	r.Read(&a_type)
	r.Read(&a_timestamp)
	r.Read(&decompressed_size)

	gc.log.Debug().Msgf("UAD: type %d time %d decompressed_size %d", a_type, a_timestamp, decompressed_size)

	if a_type >= uint32(wow.NUM_ACCOUNT_DATA_TYPES) {
		gc.log.Warn().
			Msgf("UAD: Detected account of type %d, which meets or exceeds the maximum %d",
				a_type, wow.NUM_ACCOUNT_DATA_TYPES)
		return
	}

	if decompressed_size > 0xFFFF {
		// C++ is doing recv_data.rfinish() with comment "unneeded warning spam in this case"
		gc.log.Warn().
			Msgf("UAD: Account data packet size too big. Size: %d", decompressed_size)
		return
	}

	account_type := wow.AccountDataType(a_type)

	w := wow.NewPacket(wow.ServerUpdateAccountDataComplete)

	if decompressed_size == 0 {
		gc.acc.Metadata[account_type].Data = ""
		gc.acc.Metadata[account_type].Time = 0

	} else {
		// this assumes it's a C-style string.. might not be, might need to just dump the rest?
		r.ReadString(&gc.acc.Metadata[account_type].Data)
		gc.acc.Metadata[account_type].Time = a_timestamp
	}

	w.Write(a_type)
	w.Write(uint32(0))

	gc.SendPayload(int(wow.ServerUpdateAccountDataComplete), w.Bytes())

}

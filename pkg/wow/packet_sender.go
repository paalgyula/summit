package wow

type PayloadSender interface {
	SendPayload(opcode int, payload []byte)
}

type PacketSender interface {
	Send(pkt Packet)
}

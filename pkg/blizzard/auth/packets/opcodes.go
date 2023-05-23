package packets

type Opcodes []*Opcode

func (o Opcodes) GetByID(id int) *Opcode {
	for _, v := range o {
		if v.ID == AuthCmd(id) {
			return v
		}
	}

	return nil
}

type Opcode struct {
	ID      AuthCmd
	Name    string
	Service string
	State   string
	Handler any
}

type AuthCmd uint32

const (
	AuthLoginChallenge AuthCmd = iota + 0x00
	AuthLoginProof
	AuthReconnectChallenge
	AuthReconnectProof
	RealmList AuthCmd = 0x10
)

var opcodes = Opcodes{
	{AuthLoginChallenge, "LoginChallenge", "Realm", "none", nil},
	{AuthLoginProof, "LoginProof", "Realm", "none", nil},
	{RealmList, "Realmlist", "Realm", "none", nil},
}

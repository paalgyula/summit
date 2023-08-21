package auth

type RealmProvider interface {
	// Returns a list of realms for the given account.
	Realms(accountID string) ([]*Realm, error)
}

type StaticRealmProvider struct {
	RealmList []*Realm
}

func (srp *StaticRealmProvider) Realms(_ string) ([]*Realm, error) {
	return srp.RealmList, nil
}

type RealmFlags uint8

const (
	RealmFlagNone         RealmFlags = 0x00
	RealmFlagInvalid      RealmFlags = 0x01
	RealmFlagOffline      RealmFlags = 0x02
	RealmFlagSpecifyBuild RealmFlags = 0x04
	RealmFlagUnk1         RealmFlags = 0x08
	RealmFlagUnk2         RealmFlags = 0x10
	RealmFlagNewPlayers   RealmFlags = 0x20
	RealmFlagRecommended  RealmFlags = 0x40
	RealmFlagFull         RealmFlags = 0x80
)

// Realm is information required to send as part of the realmlist.
type Realm struct {
	// realm type (this is second column in Cfg_Configs.dbc)
	Icon uint8
	// flags, if 0x01, then realm locked
	Lock uint8
	// see enum RealmFlags
	Flags RealmFlags
	// Name name of the server
	Name string
	// Address is a network address of the world server
	Address string
	// Population
	Population float32
	// NumCharacters number of characters in server
	NumCharacters uint8
	// Timezone
	Timezone uint8

	// Unknown - needs research whats this
	Unknown uint8
}

package auth

import (
	"fmt"
	"io"
	"math/big"
	"strings"

	"github.com/paalgyula/summit/pkg/wow"
)

// ClientLoginChallenge received login challenge packet
type ClientLoginChallenge struct {
	GameName        string
	Version         [3]byte
	Build           uint16
	Platform        string
	OS              string
	Locale          string
	WorldRegionBias uint32
	IP              [4]uint8
	AccountName     string
}

func NewClientLoginChallenge(accName string) *ClientLoginChallenge {
	return &ClientLoginChallenge{
		GameName:        "WoW\x00",
		Version:         [3]byte{3, 3, 5},
		Build:           12340,
		Platform:        "68x\x00",
		OS:              "niW\x00",
		Locale:          "SUne",
		WorldRegionBias: 0,
		IP:              [4]uint8{89, 51, 25, 12},
		AccountName:     strings.ToUpper(accName),
	}
}

func (p *ClientLoginChallenge) OpCode() RealmCommand {
	return AuthLoginChallenge
}

func (p *ClientLoginChallenge) UnmarshalPacket(bb wow.PacketData) error {
	r := bb.Reader()
	r.ReadStringFixed(&p.GameName, 4)
	r.ReadL(&p.Version)
	r.ReadL(&p.Build)
	r.ReadStringFixed(&p.Platform, 4)
	r.ReadStringFixed(&p.OS, 4)
	r.ReadStringFixed(&p.Locale, 4)
	r.ReadL(&p.WorldRegionBias)
	r.ReadL(&p.IP)

	var len uint8
	r.ReadB(&len)

	r.ReadStringFixed(&p.AccountName, int(len))

	return nil
}

func (p *ClientLoginChallenge) MarshalPacket() []byte {
	w := wow.NewPacket(wow.OpCode(AuthLoginChallenge))

	w.WriteStringFixed(p.GameName, 4)
	w.WriteBytes(p.Version[:])
	w.Write(p.Build)
	w.WriteStringFixed(p.Platform, 4)
	w.WriteStringFixed(p.OS, 4)
	w.WriteStringFixed(p.Locale, 4)
	w.Write(p.WorldRegionBias)
	w.Write(p.IP)

	w.Write(uint8(len(p.AccountName)))
	w.WriteStringFixed(p.AccountName, len(p.AccountName))

	return w.Bytes()
}

type ChallengeStatus = uint8

const (
	ChallengeStatusSuccess ChallengeStatus = iota
	// Unable to connect. Please try again later.
	ChallengeStatusFailed
	// This <game> account has been closed and is no longer available for use. Please
	// go to <site>/banned.html for further information.
	ChallengeStatusFailBanned ChallengeStatus = iota + 1
	// The information you have entered is not valid. Please check the spelling
	// of the account name and password. If you need help in retrieving a lost or
	// stolen password, see <site> for more information
	ChallengeStatusFailUnknownAccount
	// The information you have entered is not valid. Please check the spelling
	// of the account name and password. If you need help in retrieving a lost
	// or stolen password, see <site> for more information
	ChallengeStatusFailIncorrectPassword
	// This account is already logged into <game>. Please check the spelling and try again.
	ChallengeStatusFailAlreadyOnline
	// You have used up your prepaid time for this account. Please purchase more to continue playing
	ChallengeStatusFailNoTime
	// Could not log in to <game> at this time. Please try again later.
	ChallengeStatusFailDbBusy
	// Unable to validate game version. This may be caused by file corruption or
	// interference of another program. Please visit <site> for more information
	// and possible solutions to this issue.
	ChallengeStatusFailVersionInvalid
	// Downloading
	ChallengeStatusFailVersionUpdate
	// Unable to connect
	ChallengeStatusFailInvalidServer
	// This <game> account has been temporarily suspended. Please go to <site>/banned.html for further information
	ChallengeStatusFailSuspended
	// Unable to connect
	ChallengeStatusFailFailNoaccess
	// Connected.
	ChallengeStatusSuccessSurvey
	// Access to this account has been blocked by parental controls. Your settings may be changed in your account preferences at <site>
	ChallengeStatusFailParentcontrol
	// You have applied a lock to your account. You can change your locked status by calling your account lock phone number.
	ChallengeStatusFailLockedEnforced
	// Your trial subscription has expired. Please visit <site> to upgrade your account.
	ChallengeStatusFailTrialEnded
	// This account is now attached to a Battle.net account. Please log in with your Battle.net account email address (example: john.doe@blizzard.com) and password.
	ChallengeStatusFailUseBattleNet
	// unable to connect
	ChallengeStatusFailAntiIndulgence
	// unable to connect
	ChallengeStatusFailExpired
	// unable to connect
	ChallengeStatusFailNoGameAccount
	// This World of Warcraft account has been temporarily closed due to a chargeback on its subscription. Please refer to this [link] fo further information.
	ChallengeStatusFailChargeback
	// In order to log in to World of Warcraft using IGR time, this World of Warcraft account must first be merged with a Battle.net account. Please visit [link] to merge this account.
	ChallengeStatusFailInternetGameRoomWithoutBnet
	// Access to your account has been temporarily disabled. Please contact support for more information at: [link/account-error]
	ChallengeStatusFailGameAccountLocked
	// Your account has been locked but can be unlocked.
	ChallengeStatusFailUnlockableLock
	// You must log in with a Battle.net account username and password. TO create an account please [Click Here] or go to [link] to begin the conversion.
	ChallengeStatusFailConversionRequired ChallengeStatus = 0x20
	// You have been disconnected from the server.
	ChallengeStatusFailDisconnected ChallengeStatus = 0xFF
)

// ServerLoginChallenge is the server's response to a client's challenge. It contains
// some SRP information used for handshaking.
type ServerLoginChallenge struct {
	Status ChallengeStatus
	B      big.Int
	Salt   big.Int
	// 16 bytes long
	SaltCRC []byte

	G uint8
	N big.Int
}

func (pkt *ServerLoginChallenge) ReadPacket(data io.Reader) int {
	r := wow.NewConnectionReader(data)

	var tmp uint8

	r.Read(&tmp) // protocol versioon
	r.Read(&pkt.Status)

	if pkt.Status == ChallengeStatusSuccess {
		pkt.B.SetBytes(r.ReadReverseBytes(32))
		r.Read(&tmp) // Size of G
		r.Read(&pkt.G)

		pkt.N = big.Int{}
		r.Read(&tmp)
		pkt.N.SetBytes(r.ReadReverseBytes(int(tmp)))

		pkt.Salt.SetBytes(r.ReadReverseBytes(32))
		pkt.SaltCRC, _ = r.ReadNBytes(16)

		r.Read(&tmp)
	}

	return r.ReadedCount()
}

// Bytes writes out the packet to an array of bytes.
func (pkt *ServerLoginChallenge) MarshalPacket() []byte {
	w := wow.NewPacket(wow.OpCode(AuthLoginChallenge))

	w.WriteOne(0) // unk1
	w.WriteOne(int(pkt.Status))

	if pkt.Status == ChallengeStatusSuccess {
		// Public key of SRP6
		w.WriteZeroPadded(wow.ReverseBytes(pkt.B.Bytes()), 32)

		fmt.Println("B: ", pkt.B.Text(16))

		// G is the generator of SRP6
		w.WriteOne(0x01)
		w.Write(pkt.G)

		// Send the shared N prime
		nb := pkt.N.Bytes()
		w.WriteOne(len(nb))
		w.WriteReverseBytes(nb)

		// Salt of the password generator
		w.WriteZeroPadded(wow.ReverseBytes(pkt.Salt.Bytes()), 32)
		fmt.Println("Salt: ", pkt.Salt.Text(16))

		w.WriteBytes(pkt.SaltCRC)

		w.WriteOne(0) // unk2
	}

	return w.Bytes()
}

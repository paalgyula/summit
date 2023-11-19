package auth

import (
	"io"
	"math/big"
	"net"
	"strings"

	"github.com/paalgyula/summit/pkg/wow"
)

// ClientLoginChallenge received login challenge packet.
type ClientLoginChallenge struct {
	GameName        string
	Version         [3]byte
	Build           uint16
	Platform        string
	OS              string
	Locale          string
	WorldRegionBias uint32
	IP              []uint8
	AccountName     string
}

func NewClientLoginChallenge(accName string, build int, version [3]byte) *ClientLoginChallenge {
	ip := net.ParseIP("84.224.232.176")
	ipbytes := []uint8(ip.To4())

	return &ClientLoginChallenge{
		GameName:        "WoW\x00", // Always WoW
		Version:         version,
		Build:           uint16(build), // WOTLK
		Platform:        "68x\x00",     // x86
		OS:              "niW\x00",     // Win
		Locale:          "SUne",        // enUS
		WorldRegionBias: 0,
		IP:              ipbytes,
		AccountName:     strings.ToUpper(accName),
	}
}

func (p *ClientLoginChallenge) OpCode() RealmCommand {
	return AuthLoginChallenge
}

//nolint:errcheck
func (p *ClientLoginChallenge) UnmarshalPacket(bb wow.PacketData) error {
	r := bb.Reader()
	r.ReadStringFixed(&p.GameName, 4)
	r.ReadL(&p.Version)
	r.ReadL(&p.Build)
	r.ReadStringFixed(&p.Platform, 4)
	r.ReadStringFixed(&p.OS, 4)
	r.ReadStringFixed(&p.Locale, 4)
	r.ReadL(&p.WorldRegionBias)
	p.IP, _ = r.ReadNBytes(4)

	var size uint8
	_ = r.ReadB(&size)

	r.ReadStringFixed(&p.AccountName, int(size))

	return nil
}

//nolint:errcheck
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

type ChallengeStatus uint8

const (
	ChallengeStatusSuccess ChallengeStatus = iota
	// Account not found.
	ChallengeStatusFailed
	// Account has been banned.
	ChallengeStatusFailBanned
	// This <game> account has been closed and is no longer available for use. Please
	// go to <site>/banned.html for further information.
	ChallengeStatusFailUnknownAccount
	// The information you have entered is not valid. Please check the spelling
	// of the account name and password. If you need help in retrieving a lost or
	// stolen password, see <site> for more information.
	ChallengeStatusFailUnknown0
	// The information you have entered is not valid. Please check the spelling
	// of the account name and password. If you need help in retrieving a lost
	// or stolen password, see <site> for more information.
	ChallengeStatusFailIncorrectPassword
	// This account is already logged into <game>. Please check the spelling and try again.
	ChallengeStatusFailAlreadyOnline
	// You have used up your prepaid time for this account. Please purchase more to continue playing.
	ChallengeStatusFailNoTime
	// Could not log in to <game> at this time. Please try again later.
	ChallengeStatusFailDBBusy
	// Unable to validate game version. This may be caused by file corruption or
	// interference of another program. Please visit <site> for more information
	// and possible solutions to this issue.
	ChallengeStatusFailVersionInvalid
	// Downloading.
	ChallengeStatusFailVersionUpdate
	// Unable to connect.
	ChallengeStatusFailInvalidServer
	// This <game> account has been temporarily suspended. Please go to <site>/banned.html for further information.
	ChallengeStatusFailSuspended
	// Unable to connect.
	ChallengeStatusFailFailNoaccess
	// Connected.
	ChallengeStatusSuccessSurvey
	// Access to this account has been blocked by parental controls. Your settings may be changed in your
	// account preferences or at <site>.
	ChallengeStatusFailParentcontrol
	// You have applied a lock to your account. You can change your locked status by calling
	// your account lock phone number.
	ChallengeStatusFailLockedEnforced
	// Your trial subscription has expired. Please visit <site> to upgrade your account.
	ChallengeStatusFailTrialEnded
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

//nolint:errcheck
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
		_ = r.Read(&tmp)
		pkt.N.SetBytes(r.ReadReverseBytes(int(tmp)))

		pkt.Salt.SetBytes(r.ReadReverseBytes(32))
		pkt.SaltCRC, _ = r.ReadNBytes(16)

		r.Read(&tmp)
	}

	return r.ReadedCount()
}

// Bytes writes out the packet to an array of bytes.
//
//nolint:errcheck
func (pkt *ServerLoginChallenge) MarshalPacket() []byte {
	w := wow.NewPacket(wow.OpCode(AuthLoginChallenge))

	w.WriteOne(0) // unk1
	w.WriteOne(int(pkt.Status))

	if pkt.Status == ChallengeStatusSuccess {
		// Public key of SRP6
		w.WriteZeroPadded(wow.ReverseBytes(pkt.B.Bytes()), 32)

		// Only for debug purposes
		// fmt.Println("B: ", pkt.B.Text(16))

		// G is the generator of SRP6
		w.WriteOne(0x01)
		w.Write(pkt.G)

		// Send the shared N prime
		nb := pkt.N.Bytes()
		w.WriteOne(len(nb))
		w.WriteReverseBytes(nb)

		// Salt of the password generator
		w.WriteZeroPadded(wow.ReverseBytes(pkt.Salt.Bytes()), 32)
		// fmt.Println("Salt: ", pkt.Salt.Text(16))

		w.WriteBytes(pkt.SaltCRC)

		w.WriteOne(0) // unk2
	}

	return w.Bytes()
}

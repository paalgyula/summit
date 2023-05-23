package packets

import (
	"math/big"

	"github.com/paalgyula/summit/lib/util"
	"github.com/paalgyula/summit/pkg/blizzard/auth/srp"
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

func (p *ClientLoginChallenge) UnmarshalPacket(bb []byte) error {
	r := wow.NewPacketReader(bb)
	p.GameName = r.ReadStringFixed(4)
	r.ReadL(&p.Version)
	r.ReadL(&p.Build)
	p.Platform = r.ReadStringFixed(4)
	p.OS = r.ReadStringFixed(4)
	p.Locale = r.ReadStringFixed(4)
	r.ReadL(&p.WorldRegionBias)
	r.ReadL(&p.IP)

	var len uint8
	r.ReadB(&len)

	p.AccountName = r.ReadStringFixed(int(len))

	return nil
}

func (p *ClientLoginChallenge) MarshalPacket() []byte {
	w := wow.NewPacketWriter()
	w.WriteString(p.GameName)
	w.Write(p.Version[:])
	w.WriteL(p.Build)
	w.WriteString(p.Platform)
	w.WriteString(p.OS)
	w.WriteString(p.Locale)
	w.WriteL(p.WorldRegionBias)
	w.WriteL(p.IP)

	w.WriteL(uint8(len(p.AccountName)))
	w.Write([]byte(p.AccountName))

	return w.Bytes()
}

type ChallengeStatus uint8

const (
	ChallengeStatusSuccess ChallengeStatus = iota
	// Account not found
	ChallengeStatusFailed
	// Account has been banned
	ChallengeStatusFailBanned
	// This <game> account has been closed and is no longer available for use. Please go to <site>/banned.html for further information.
	ChallengeStatusFailUnknownAccount
	// The information you have entered is not valid. Please check the spelling of the account name and password. If you need help in retrieving a lost or stolen password, see <site> for more information
	ChallengeStatusFailUnknown0
	ChallengeStatusFailIncorrectPassword ///< The information you have entered is not valid. Please check the spelling of the account name and password. If you need help in retrieving a lost or stolen password, see <site> for more information
	ChallengeStatusFailAlreadyOnline     ///< This account is already logged into <game>. Please check the spelling and try again.
	ChallengeStatusFailNoTime            ///< You have used up your prepaid time for this account. Please purchase more to continue playing
	ChallengeStatusFailDbBusy            ///< Could not log in to <game> at this time. Please try again later.
	ChallengeStatusFailVersionInvalid    ///< Unable to validate game version. This may be caused by file corruption or interference of another program. Please visit <site> for more information and possible solutions to this issue.
	ChallengeStatusFailVersionUpdate     ///< Downloading
	ChallengeStatusFailInvalidServer     ///< Unable to connect
	ChallengeStatusFailSuspended         ///< This <game> account has been temporarily suspended. Please go to <site>/banned.html for further information
	ChallengeStatusFailFailNoaccess      ///< Unable to connect
	ChallengeStatusSuccessSurvey         ///< Connected.
	ChallengeStatusFailParentcontrol     ///< Access to this account has been blocked by parental controls. Your settings may be changed in your account preferences at <site>
	ChallengeStatusFailLockedEnforced    ///< You have applied a lock to your account. You can change your locked status by calling your account lock phone number.
	ChallengeStatusFailTrialEnded        ///< Your trial subscription has expired. Please visit <site> to upgrade your account.
)

// ServerLoginChallenge is the server's response to a client's challenge. It contains
// some SRP information used for handshaking.
type ServerLoginChallenge struct {
	Status  ChallengeStatus
	B       big.Int
	Salt    big.Int
	SaltCRC big.Int
}

// Bytes writes out the packet to an array of bytes.
func (pkt *ServerLoginChallenge) MarshalPacket() []byte {
	w := wow.NewPacketWriter()

	w.WriteByte(0) // unk1
	w.WriteByte(uint8(pkt.Status))

	if pkt.Status == ChallengeStatusSuccess {
		w.Write(util.PadBigIntBytes(util.ReverseBytes(pkt.B.Bytes()), 32))
		w.WriteByte(1)
		w.WriteByte(srp.G)
		w.WriteByte(32)
		w.WriteReverse(srp.N().Bytes())
		w.Write(util.PadBigIntBytes(util.ReverseBytes(pkt.Salt.Bytes()), 32))
		w.Write(util.PadBigIntBytes(util.ReverseBytes(pkt.SaltCRC.Bytes()), 16))
		w.WriteByte(0) // unk2
	}

	return w.Bytes()
}

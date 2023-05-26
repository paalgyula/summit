package crypt

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateEphemeralPair(t *testing.T) {
	var verifier, expectedb, expectedB big.Int
	var N big.Int
	N.SetString("62100066509156017342069496140902949863249758336000796928566441170293728648119", 10)

	s := NewSRP6(7, 3, &N)

	verifier.SetString("3e3f49a5a14a43b870f8de5534e318c63394738c364a71f205a8ba277bb56ff6", 16)

	expectedb.SetString("a4fa7a0d8b9d55f81fa3af7f386de83bbd84fa", 16)
	expectedB.SetString("700287a1578669b5de438afff3e8927ed436195969f999bc50dc1af9da4a94f1", 16)

	_ = s

	// private, public := s.GenerateEphemeralPair(&verifier)

	// fmt.Printf("B: %s\nb: %s\n", public.Text(16), private.Text(16))

	// assert.Equal(t, expectedb.Text(16), private.Text(16))
	// assert.Equal(t, expectedB.Text(16), public.Text(16))
}

func TestGenerateVerifier(t *testing.T) {
	var salt, expectedV big.Int

	var N big.Int
	N.SetString("62100066509156017342069496140902949863249758336000796928566441170293728648119", 10)

	srp6 := NewSRP6(7, 3, &N)

	salt.SetString("9398c11e0e7128c7a56e3fde45b418744ffe9c7f41aaed48ac27e62d3700e223", 16)
	expectedV.SetString("3e3f49a5a14a43b870f8de5534e318c63394738c364a71f205a8ba277bb56ff6", 16)

	v := srp6.GenerateVerifier("TEST", "TEST", &salt)

	// fmt.Printf("%s", v.Text(16))

	assert.Equal(t, v.Cmp(&expectedV), 0)
}

func TestCalculateSessionKey(t *testing.T) {
	var A, v, s, expectedK, expectedM big.Int
	var N big.Int
	N.SetString("62100066509156017342069496140902949863249758336000796928566441170293728648119", 10)

	srp6 := NewSRP6(7, 3, &N)

	A.SetString("1234344069974946706941181551060269688256096998192437644043961152849307948728", 10)
	srp6.GenerateServerPubKey(big.NewInt(0))

	srp6.B.SetString("16630279820182697578309394812726193457375869535456855997552735653810818403718", 10)
	srp6.b.SetString("3679141816495610969398422835318306156547245306", 10)

	// TEST:TEST
	v.SetString("3e3f49a5a14a43b870f8de5534e318c63394738c364a71f205a8ba277bb56ff6", 16)
	s.SetString("9398c11e0e7128c7a56e3fde45b418744ffe9c7f41aaed48ac27e62d3700e223", 16)

	expectedK.SetString("3e995d1c002c22e7e733513fc861b49ede9c285b008891d186256bd0b595bc67b941a3d8273bc828", 16)
	expectedM.SetString("e99d32d27dfe0553ac1d1558d112bb30b8d1999d", 16)

	K, M := srp6.CalculateServerSessionKey(&A, &v, &s, "TEST")

	assert.Equal(t, K.Cmp(&expectedK), 0)
	assert.Equal(t, M.Cmp(&expectedM), 0)
}

func TestCalculateServerProof(t *testing.T) {
	var A, M, K, expectedProof big.Int

	A.SetString("1234344069974946706941181551060269688256096998192437644043961152849307948728", 10)
	M.SetString("1278405643266187066239549723718271591736372958987", 10)
	K.SetString("1223778727786691224255566132121120158338166041153346746306820190174949498228440143950889596323712", 10)
	expectedProof.SetString("1284245613498486112994244042115912960631626548879", 10)

	proof := CalculateServerProof(&A, &M, &K)

	assert.Equal(t, proof.Cmp(&expectedProof), 0)
}

func TestFullPasswordlessFlow(t *testing.T) {
	// These two are shared
	var N big.Int
	N.SetString("62100066509156017342069496140902949863249758336000796928566441170293728648119", 10)

	username := "test"
	password := "belabacsi"

	I := strings.ToUpper(username)

	sClient := NewSRP6(7, 3, &N)
	sServer := NewSRP6(7, 3, &N)

	//1. server generate salt (s), and verifier (v) generation
	sSalt := sServer.RandomSalt()
	// sSalt := big.NewInt(0)
	// sSalt.SetString("9398c11e0e7128c7a56e3fde45b418744ffe9c7f41aaed48ac27e62d3700e223", 16)

	sVerifier := sServer.GenerateVerifier(I, password, sSalt)

	fmt.Printf("s: 0x%x\nv: 0x%x\n\n", sSalt, sVerifier)

	// 1. client sends username I and public ephemeral value A to the server
	A := sClient.GenerateClientPubkey()
	fmt.Printf("I: %s a: 0x%x\nA: 0x%x\n\tclient->server (I, A)\n\n", username, sClient.a, A)

	B := sServer.GenerateServerPubKey(sVerifier)
	fmt.Printf("s: 0x%x\nb: 0x%x\nB: 0x%x\n\tserver->client (s, B)\n\n", sSalt, sServer.b, B)

	// Server and client exchanges A, B, I
	uS := sServer.RandomScrambling(A, B)
	uC := sClient.RandomScrambling(A, B)
	fmt.Printf("u: 0x%x\n   0x%x # scrambling\n\n", uS, uC)

	sK, sM := sServer.CalculateServerSessionKey(A, sVerifier, sSalt, I)
	cK, cM := sClient.CalculateClientSessionKey(sSalt, B, I, password)
	_ = cM
	fmt.Printf("sK: 0x%x\ncK: 0x%x\nsM: 0x%x\ncM: 0x%x\n\n", sK, cK, sM, cM)

	assert.Equal(t, sK.Text(16), cK.Text(16))
	assert.Equal(t, sM.Text(16), cM.Text(16))
}

func TestNLength(t *testing.T) {
	srp := NewSRP6(7, 3, big.NewInt(0))
	assert.Len(t, srp.N().Bytes(), 32)
}

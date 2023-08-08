// The Secure Remote Password protocol (SRP) is an augmented password-authenticated key exchange (PAKE) protocol,
// specifically designed to work around existing patents.
package crypt

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"math/big"
)

type SRP6 struct {
	// G is the SRP Generator; the base of many mathematical expressions.
	g int64

	// K is the SRP Verifier Scale Factor; used to scale the verifier which
	// is stored in the database.
	//
	// 	k = H(N, g) // Multiplier parameter (k=3 in legacy SRP-6)
	//
	k int64

	// N is the SRP Modulus; all operations are performed in base N.
	n *big.Int

	// Client private (a) and public (A) key
	A, a *big.Int

	// Server private (b) and public (B) key
	B, b *big.Int
}

// VALE_QUESTION: What is the purpose here of the parameters g & k? They aren't used. Even N seems redundant in how you've used it to-date..

// NewSRP6 g=7 k=3 N=bignumber
func NewSRP6(g, k int64, N *big.Int) *SRP6 {
	srp6 := &SRP6{
		g: int64(7),
		k: int64(3),
		n: N,
	}

	// 894B645E89E1535BBDAD5B8B290650530801B18EBFBF5E8FAB3C82872A3E9BB7
	srp6.n.SetString("62100066509156017342069496140902949863249758336000796928566441170293728648119", 10)

	return srp6
}

func (s *SRP6) GValue() int64 {
	return s.g
}

func (s *SRP6) G() *big.Int {
	return big.NewInt(s.g)
}

func (s *SRP6) RandomSalt() *big.Int {
	// Generate a random big number with 256 bits
	randomBits := 256

	// Generate random bytes
	randomBytes := make([]byte, randomBits/8)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}

	// Convert random bytes to a big.Int
	randomNumber := new(big.Int)
	randomNumber.SetBytes(randomBytes)

	return randomNumber
}

// N is the SRP Modulus; all operations are performed in base N.
func (s *SRP6) N() *big.Int {
	return s.n
}

func Hash(parts ...[]byte) []byte {
	hash := sha1.New()
	for _, part := range parts {
		hash.Write(reverse(part))
	}

	return reverse(hash.Sum(nil))
}

// GenerateVerifier will generate a hash of the account name, password and salt
// which can be used as the SRP verifier.
func (s *SRP6) GenerateVerifier(accountName, password string, salt *big.Int) *big.Int {
	x := big.NewInt(0)
	// x = H(s, I, p) => H(s, (I|:|p))
	x.SetBytes(Hash(salt.Bytes(), Hash(reverse(
		[]byte(fmt.Sprintf("%s:%s", accountName, password))),
	)))

	g := big.NewInt(int64(s.g))
	return g.Exp(g, x, s.N())
}

// generatePublicEphemeral calculaes the B value given b & v.
func (s *SRP6) generatePublicEphemeral(v *big.Int, b *big.Int) *big.Int {
	g := big.NewInt(int64(s.g))

	B := big.NewInt(0)
	B.Mul(v, big.NewInt(s.k))
	B.Add(B, g.Exp(g, b, s.N()))
	B.Mod(B, s.N())

	return B
}

// GenerateServerPubKey generates a public ephemeral pair (B, b) given a user's verifier.
// The private key stored in this instance for later use
func (s *SRP6) GenerateServerPubKey(v *big.Int) *big.Int {
	s.b = s.RandomSalt()
	s.B = s.generatePublicEphemeral(v, s.b)

	return s.B
}

func (s *SRP6) GenerateClientPubkey() *big.Int {
	s.a = s.RandomSalt()
	// a := big.NewInt(0)
	// // 894B645E89E1535BBDAD5B8B290650530801B18EBFBF5E8FAB3C82872A3E9BB7
	// a.SetString("62100066509156017342069496140902949863249758336000796928566441170293728648119", 10)
	// s.a = a

	s.A = new(big.Int).Exp(big.NewInt(s.g), s.a, s.N())

	return s.A
}

func (s *SRP6) RandomScrambling(A, B *big.Int) *big.Int {
	u := big.NewInt(0)
	u.SetBytes(Hash(A.Bytes(), B.Bytes()))

	return u
}

func padBigIntBytes(data []byte, nBytes int) []byte {
	if len(data) > nBytes {
		return data[:nBytes]
	}

	currSize := len(data)
	for i := 0; i < nBytes-currSize; i++ {
		data = append([]byte{'\x00'}, data...)
	}

	return data
}

func interleave(S *big.Int) *big.Int {
	T := padBigIntBytes(reverse(S.Bytes()), 32)

	G := make([]byte, 16)
	H := make([]byte, 16)

	for i := 0; i < 16; i++ {
		G[i] = T[i*2]
		H[i] = T[i*2+1]
	}

	G = reverse(Hash(reverse(G)))
	H = reverse(Hash(reverse(H)))

	K := make([]byte, 0)
	for i := 0; i < 20; i++ {
		K = append(K, G[i], H[i])
	}

	KInt := big.NewInt(0)
	KInt.SetBytes(reverse(K))

	return KInt
}

func reverse(data []byte) []byte {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}

	return data
}

func (s *SRP6) CalculateClientSessionKey(salt, B *big.Int, I, P string) (*big.Int, *big.Int) {
	u := s.RandomScrambling(s.A, B)

	x := big.NewInt(0)
	// x = H(salt, I, p) => H(salt, (I|:|p))
	x.SetBytes(Hash(salt.Bytes(), Hash(reverse(
		[]byte(fmt.Sprintf("%s:%s", I, P))),
	)))

	// S_c = pow(B - k * pow(g, x, N), a + u * x, N)
	S := big.NewInt(0)
	S.Exp(big.NewInt(s.g), x, s.N())
	S.Mul(S, big.NewInt(s.k))
	S.Sub(B, S)

	uxa := big.NewInt(0)
	uxa.Mul(u, x)
	uxa.Add(s.a, uxa)

	S.Exp(S, uxa, s.N())

	K := interleave(S)

	NHash := big.NewInt(0)
	NHash.SetBytes(Hash(s.N().Bytes()))

	gHash := big.NewInt(0)
	gHash.SetBytes(Hash(big.NewInt(s.g).Bytes()))
	gHash.Xor(gHash, NHash)

	M := big.NewInt(0)
	M.SetBytes(
		Hash(gHash.Bytes(), Hash(
			reverse([]byte(I))), // I
			salt.Bytes(), // s
			s.A.Bytes(),  // A
			B.Bytes(),    // B
			K.Bytes(),    // K
		),
	)

	return K, M
}

// CalculateServerSessionKey takes as input the client's proof and calculates the
// persistent session key.
func (s *SRP6) CalculateServerSessionKey(A, v, salt *big.Int, accountName string) (*big.Int, *big.Int) {
	u := s.RandomScrambling(A, s.B)

	// S_s = pow(A * pow(v, u, N), b, N)
	// K_s = H(S_s)
	S := big.NewInt(0)
	S.Exp(v, u, s.N())
	S.Mul(S, A)
	S.Exp(S, s.b, s.N())

	K := interleave(S)

	NHash := big.NewInt(0)
	NHash.SetBytes(Hash(s.N().Bytes()))

	gHash := big.NewInt(0)
	gHash.SetBytes(Hash(big.NewInt(s.g).Bytes()))
	gHash.Xor(gHash, NHash)

	M := big.NewInt(0)
	M.SetBytes(
		Hash(gHash.Bytes(), Hash(
			reverse([]byte(accountName))), // I
			salt.Bytes(), // s
			A.Bytes(),    // A
			s.B.Bytes(),  // B
			K.Bytes(),    // K
		),
	)

	return K, M
}

// CalculateServerProof will calculate a proof to send back to the client so they
// know we are a legit server.
func CalculateServerProof(A, M, K *big.Int) *big.Int {
	proof := big.NewInt(0)
	proof.SetBytes(Hash(A.Bytes(), M.Bytes(), K.Bytes()))
	return proof
}

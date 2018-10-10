// Package pedersen implements a Verifiable Secret Sharing (VSS) scheme described
// by Torben Pryds Pedersen in the referenced [Ped91b] paper.
// It consists of VSS parameters structure and functions to calculate and verify
// a commitment to chosen value.
//
// Commitment scheme allows a party (Commiter) to commit to a chosen value while
// keeping the value hidden from the other party (Verifier).
// On verification stage Committer reveals the value along with a DecommitmentKey,
// so Verifier can confirm the revealed value matches the committed one.
//
// pedersen.NewVSS() initializes scheme with `g` and `h` values, which need to
// be randomly generated for each scheme execution.
// To stop an adversary Committer from changing the value them already committed
// to, the scheme requires that `log_g(h)` is unknown to the Committer.
//
// You may consult our documentation for more details:
// docs/cryptography/trapdoor-commitments.html#_pedersen_commitment
//
//     [Ped91b]: T. Pedersen. Non-interactive and information-theoretic secure
//         verifiable secret sharing. In: Advances in Cryptology — Crypto '91,
//         pages 129-140. LNCS No. 576.
//         https://www.cs.cornell.edu/courses/cs754/2001fa/129.PDF
//     [GJKR 99]: Gennaro R., Jarecki S., Krawczyk H., Rabin T. (1999) Secure
//         Distributed Key Generation for Discrete-Log Based Cryptosystems. In:
//         Stern J. (eds) Advances in Cryptology — EUROCRYPT ’99. EUROCRYPT 1999.
//         Lecture Notes in Computer Science, vol 1592. Springer, Berlin, Heidelberg
//         http://groups.csail.mit.edu/cis/pubs/stasio/vss.ps.gz
package pedersen

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/keep-network/keep-core/pkg/internal/byteutils"
)

// VSS scheme parameters
type VSS struct {
	// g and h are elements of a group of order q, and should be chosen such that
	// no one knows log_g(h).
	g, h *big.Int
}

// Commitment represents a single commitment to a single message. One is produced
// for each message we have committed to.
//
// It is usually shared with the verifier immediately after it has been produced
// and lets the recipient verify if the message revealed later by the committing
// party is really what that party has committed to.
//
// The commitment itself is not enough for a verification. In order to perform
// a verification, the interested party must receive the `DecommitmentKey`.
type Commitment struct {
	vss        *VSS
	commitment *big.Int
}

// DecommitmentKey represents the key that allows a recipient to open an
// already-received commitment and verify if the value is what the sender have
// really committed to.
type DecommitmentKey struct {
	r *big.Int
}

// Primes such that `p = 2q + 1`.
var p, q *big.Int

func init() {
	// Sets p and q to predefined fixed values, such that `p = 2q + 1`.
	// `p` is 4096-bit safe prime.
	pStr := "0xc8526644a9c4739683742b7003640b2023ca42cc018a42b02a551bb825c6828f86e2e216ea5d31004c433582a3fa720459efb42e091d73fb281810e1825691f0799811be62ae57f62ab00670edd35426d108d3b9c4fd008eddc67275a0489fe132e4c31bd7069ea7884cbb8f8f9255fe7b87fc0099f246776c340912df48f7945bc2bc0bc6814978d27b7af2ebc41f458ae795186db0fd7e6151bb8a7fe2b41370f7a2848ef75d3ec88f3439022c10e78b434c2f24b2f40bd02930e6c8aadef87b0dc87cdba07dcfa86884a168bd1381a4f48be12e5d98e41f954c37aec011cc683570e8890418756ed98ace8c8e59ae1df50962c1622fe66b5409f330cad6b7c68f2e884786d9807190b89ac4a3b3507e49b2dd3f33d765ad29e2015180c8cd0258dd8bdaab17be5d74871fec04c492240c6a2692b2c9a62c9adbaac34a333f135801ff948e8dfb6bbd6212a67950fb8edd628d05d19d1b94e9be7c52ed484831d50adaa29e71de197e351878f1c40ec67ee809e824124529e27bd5ecf3054f6784153f7db27ff0c87420bb2b2754ed363fc2ba8399d49d291f342173e7619183467a9694efa243e1d41b26c13b38ca0f43bb7c9050eb966461f28436583a9d13d2c1465b78184eae360f009505ccea288a053d111988d55c12befd882a857a530efac2c0592987cd83c39844a10e058739ab1c39006a3123e7fc887845675f"
	// `q` is 4095-bit Sophie Germain prime.
	qStr := "0x6429332254e239cb41ba15b801b2059011e5216600c52158152a8ddc12e34147c371710b752e988026219ac151fd39022cf7da17048eb9fd940c0870c12b48f83ccc08df31572bfb1558033876e9aa13688469dce27e80476ee3393ad0244ff09972618deb834f53c4265dc7c7c92aff3dc3fe004cf9233bb61a04896fa47bca2de15e05e340a4bc693dbd7975e20fa2c573ca8c36d87ebf30a8ddc53ff15a09b87bd142477bae9f64479a1c81160873c5a1a61792597a05e814987364556f7c3d86e43e6dd03ee7d4344250b45e89c0d27a45f0972ecc720fcaa61bd76008e6341ab87444820c3ab76cc56746472cd70efa84b160b117f335aa04f998656b5be347974423c36cc038c85c4d6251d9a83f24d96e9f99ebb2d694f100a8c06466812c6ec5ed558bdf2eba438ff602624912063513495964d3164d6dd561a5199f89ac00ffca4746fdb5deb109533ca87dc76eb14682e8ce8dca74df3e2976a42418ea856d514f38ef0cbf1a8c3c78e207633f7404f412092294f13deaf67982a7b3c20a9fbed93ff8643a105d9593aa769b1fe15d41ccea4e948f9a10b9f3b0c8c1a33d4b4a77d121f0ea0d93609d9c6507a1ddbe482875cb3230f9421b2c1d4e89e960a32dbc0c27571b07804a82e6751445029e888cc46aae095f7ec41542bd29877d61602c94c3e6c1e1cc22508702c39cd58e1c80351891f3fe443c22b3af"

	var result bool

	p, result = new(big.Int).SetString(pStr, 0)
	if !result {
		panic("failed to initialize p")
	}

	q, result = new(big.Int).SetString(qStr, 0)
	if !result {
		panic("failed to initialize q")
	}
}

// NewVSS generates parameters for a scheme execution
func NewVSS() (*VSS, error) {
	randomG, err := randomFromZn(p)
	if err != nil {
		return nil, fmt.Errorf("g generation failed [%s]", err)
	}
	g := new(big.Int).Exp(randomG, big.NewInt(2), nil) // (randomZ(0, 2^p - 1]) ^2

	// Generate `h` jointly by the players as described in section 4.2 of [GJKR 99]
	// First players have to jointly generate a random value r ∈ Z*_p with coin
	// flipping protocol.
	// To generate a random element `h` in a subgroup generated by `g` one needs
	// to calculate `h = r^k mod p` where `k = (p - 1) / q`
	randomValue, err := randomFromZn(p) // TODO this should be generated with coin flipping protocol
	if err != nil {
		return nil, fmt.Errorf("randomValue generation failed [%s]", err)
	}

	k := new(big.Int).Div(
		new(big.Int).Sub(p, big.NewInt(1)),
		q,
	)

	h := new(big.Int).Exp(randomValue, k, p)

	return &VSS{g: g, h: h}, nil
}

// CommitmentTo takes a secret message and a set of parameters and returns
// a commitment to that message and the associated decommitment key.
//
// First random `r` value is chosen as a Decommitment Key.
// Then commitment is calculated as `(g ^ digest) * (h ^ r) mod p`, where digest
// is sha256 hash of the secret brought to big.Int.
func (vss *VSS) CommitmentTo(secret []byte) (*Commitment, *DecommitmentKey, error) {
	r, err := randomFromZn(q) // randomZ(0, 2^q - 1]
	if err != nil {
		return nil, nil, fmt.Errorf("r generation failed [%s]", err)
	}

	digest := hashBytesToBigInt(secret, q)
	commitment := CalculateCommitment(vss, digest, r)

	return &Commitment{vss, commitment},
		&DecommitmentKey{r},
		nil
}

// Verify checks the received commitment against the revealed secret message.
func (c *Commitment) Verify(decommitmentKey *DecommitmentKey, secret []byte) bool {
	digest := hashBytesToBigInt(secret, q)
	expectedCommitment := CalculateCommitment(c.vss, digest, decommitmentKey.r)
	return expectedCommitment.Cmp(c.commitment) == 0
}

func hashBytesToBigInt(secret []byte, mod *big.Int) *big.Int {
	hash := byteutils.Sha256Sum(secret)
	digest := new(big.Int).Mod(hash, mod)
	return digest
}

// CalculateCommitment calculates a commitment with equation `(g ^ s) * (h ^ r) mod p`
// where:
// - `g` and `h` are scheme specific parameters passed in vss,
// - `s` is a message to which one is committing,
// - `r` is a decommitment key.
func CalculateCommitment(vss *VSS, digest, r *big.Int) *big.Int {
	return new(big.Int).Mod(
		new(big.Int).Mul(
			new(big.Int).Exp(vss.g, digest, p),
			new(big.Int).Exp(vss.h, r, p),
		),
		p,
	)
}

// randomFromZn generates a random `big.Int` in a range (0, 2^n - 1]
func randomFromZn(n *big.Int) (*big.Int, error) {
	x := big.NewInt(0)
	var err error
	// TODO check if this is what we really need for g,h and r
	// 2^n - 1
	max := new(big.Int).Sub(
		// new(big.Int).Exp(big.NewInt(2), n, nil),
		n,
		big.NewInt(1),
	)
	for x.Sign() == 0 {
		x, err = rand.Int(rand.Reader, max)
		if err != nil {
			return nil, fmt.Errorf("failed to generate random number [%s]", err)
		}
	}
	return x, nil
}

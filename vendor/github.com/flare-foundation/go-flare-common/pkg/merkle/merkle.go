package merkle

import (
	"errors"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ErrEmptyTree    = errors.New("empty tree")
	ErrInvalidIndex = errors.New("invalid index")
	ErrHashNotFound = errors.New("hash not found")
)

// Merkle Tree implementation with helper functions.
type Tree []common.Hash

// NewFromHex creates a new Merkle tree from the given hex values. It is
// required that the values are sorted as strings.
func NewFromHex(hexValues []string) Tree {
	values := make(Tree, len(hexValues))

	for i, hexValue := range hexValues {
		values[i] = common.HexToHash(hexValue)
	}

	return values
}

// Given an array of leaf hashes, builds the Merkle tree.
func Build(hashes []common.Hash, initialHash bool) Tree {
	if initialHash {
		hashes = mapSingleHash(hashes)
	}

	n := len(hashes)

	if n == 0 {
		return Tree{}
	}

	// Hashes must be sorted to enable binary search.
	sortedHashes := make([]common.Hash, n)
	copy(sortedHashes, hashes)
	sort.Slice(sortedHashes, func(i, j int) bool {
		return sortedHashes[i].Hex() < sortedHashes[j].Hex()
	})

	tree := make([]common.Hash, n-1, (2*n)-1)
	tree = append(tree, sortedHashes...)

	for i := n - 2; i >= 0; i-- {
		tree[i] = SortedHashPair(tree[2*i+1], tree[2*i+2])
	}

	return tree
}

// Given an array of hex-encoded leaf hashes, builds the Merkle tree.
func BuildFromHex(hexValues []string, initialHash bool) Tree {
	var hashes []common.Hash
	for i := range hexValues {
		if i == 0 || hexValues[i] != hexValues[i-1] {
			hashes = append(hashes, common.HexToHash(hexValues[i]))
		}
	}

	return Build(hashes, initialHash)
}

func mapSingleHash(hashes []common.Hash) []common.Hash {
	output := make([]common.Hash, len(hashes))

	for i := range hashes {
		output[i] = crypto.Keccak256Hash(hashes[i].Bytes())
	}

	return output
}

// SortedHashPair returns a sorted hash of two hashes.
func SortedHashPair(x, y common.Hash) common.Hash {
	if x.Hex() <= y.Hex() {
		return crypto.Keccak256Hash(x.Bytes(), y.Bytes())
	}

	return crypto.Keccak256Hash(y.Bytes(), x.Bytes())
}

// Root returns the Merkle root of the tree.
func (t Tree) Root() (common.Hash, error) {
	if len(t) == 0 {
		return common.Hash{}, ErrEmptyTree
	}

	return t[0], nil
}

// LeavesCount returns the number of leaves in the tree.
func (t Tree) LeavesCount() int {
	if len(t) == 0 {
		return 0
	}

	return (len(t) + 1) / 2
}

// Leaves returns all leaves in a slice.
func (t Tree) Leaves() []common.Hash {
	numLeaves := t.LeavesCount()
	if numLeaves == 0 {
		return nil
	}

	return t[numLeaves-1:]
}

// GetLeaf returns the `i`th leaf.
func (t Tree) GetLeaf(i int) (common.Hash, error) {
	numLeaves := t.LeavesCount()
	if numLeaves == 0 || i < 0 || i >= numLeaves {
		return common.Hash{}, ErrInvalidIndex
	}

	pos := len(t) - numLeaves + i
	return t[pos], nil
}

// GetProof returns the Merkle proof for the `i`th leaf.
func (t Tree) GetProof(i int) ([]common.Hash, error) {
	numLeaves := t.LeavesCount()
	if numLeaves == 0 || i < 0 || i >= numLeaves {
		return nil, ErrInvalidIndex
	}

	var proof []common.Hash

	for pos := len(t) - numLeaves + i; pos > 0; pos = parent(pos) {
		sibling := pos + ((2 * (pos % 2)) - 1)
		proof = append(proof, t[sibling])
	}

	return proof, nil
}

// parent returns the index of the parent node.
func parent(i int) int {
	return (i - 1) / 2
}

// GetProofFromHash returns the proof that hash is stored in a leaf of the tree.
// Returns nil if hash is not stored in the tree.
func (t Tree) GetProofFromHash(hash common.Hash) ([]common.Hash, error) {
	i, err := t.binarySearch(hash)
	if err != nil {
		return nil, err
	}

	return t.GetProof(i)
}

func (t Tree) binarySearch(hash common.Hash) (int, error) {
	leaves := t.Leaves()
	i := sort.Search(len(leaves), func(i int) bool {
		return leaves[i].Hex() >= hash.Hex()
	})

	if i < len(leaves) && leaves[i] == hash {
		return i, nil
	}

	return 0, ErrHashNotFound
}

// VerifyProof verifies a Merkle proof for a given leaf against the Merkle root.
func VerifyProof(leaf common.Hash, proof []common.Hash, root common.Hash) bool {
	hash := leaf
	for _, pair := range proof {
		hash = SortedHashPair(pair, hash)
	}

	return hash == root
}

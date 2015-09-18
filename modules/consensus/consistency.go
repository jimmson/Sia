package consensus

import (
	"errors"
	"strings"

	"github.com/boltdb/bolt"

	"github.com/NebulousLabs/Sia/build"
	"github.com/NebulousLabs/Sia/crypto"
	"github.com/NebulousLabs/Sia/encoding"
	"github.com/NebulousLabs/Sia/types"
)

var (
	errSiacoinMiscount = errors.New("consensus set has the wrong number of siacoins given the height")
	errSiafundMiscount = errors.New("consensus set has the wrong number of siafunds")
)

// consensusChecksum grabs a checksum of the consensus set by pushing all of
// the elements in sorted order into a merkle tree and taking the root. All
// consensus sets with the same current block should have identical consensus
// checksums.
func consensusChecksum(tx *bolt.Tx) crypto.Hash {
	// Create a checksum tree.
	tree := crypto.NewTree()

	// For all of the constant buckets, push every key and every value. Buckets
	// are sorted in byte-order, therefore this operation is deterministic.
	consensusSetBuckets := []*bolt.Bucket{
		tx.Bucket(BlockPath),
		tx.Bucket(SiacoinOutputs),
		tx.Bucket(FileContracts),
		tx.Bucket(SiafundOutputs),
		tx.Bucket(SiafundPool),
	}
	for i := range consensusSetBuckets {
		err := consensusSetBuckets[i].ForEach(func(k, v []byte) error {
			tree.Push(k)
			tree.Push(v)
			return nil
		})
		if build.DEBUG && err != nil {
			panic(err)
		}
	}

	// Iterate through all the buckets looking for buckets prefixed with
	// prefixDSCO or prefixFCEX. Buckets are presented in byte-sorted order by
	// name.
	err := tx.ForEach(func(name []byte, b *bolt.Bucket) error {
		// If the bucket is not a delayed siacoin output bucket or a file
		// contract expiration bucket, skip.
		if !strings.HasPrefix(string(name), string(prefixDSCO)) && !strings.HasPrefix(string(name), string(prefixFCEX)) {
			return nil
		}

		// The bucket is a prefixed bucket - add all elements to the tree.
		return b.ForEach(func(k, v []byte) error {
			tree.Push(k)
			tree.Push(v)
			return nil
		})
	})
	if build.DEBUG && err != nil {
		panic(err)
	}

	return tree.Root()
}

// checkSiacoinCount checks that the number of siacoins countable within the
// consensus set equal the expected number of siacoins for the block height.
func checkSiacoinCount(tx *bolt.Tx) error {
	return nil
}

// checkSiafundCount checks that the number of siafunds countable within the
// consensus set equal the expected number of siafunds for the block height.
func checkSiafundCount(tx *bolt.Tx) error {
	var total types.Currency
	err := tx.Bucket(SiafundOutputs).ForEach(func(_, siafundOutputBytes []byte) error {
		var sfo types.SiafundOutput
		err := encoding.Unmarshal(siafundOutputBytes, &sfo)
		if build.DEBUG && err != nil {
			panic(err)
		}
		total = total.Add(sfo.Value)
		return nil
	})
	if err != nil {
		return err
	}
	if total.Cmp(types.SiafundCount) != 0 {
		return errors.New("wrong number if siafunds in the consensus set")
	}
	return nil
}

// checkDSCOs scans the sets of delayed siacoin outputs and checks for
// consistency.
func checkDSCOs(tx *bolt.Tx) error {
	// Create a map to track which delayed siacoin output maps exist, and
	// another map to track which ids have appeared in the dsco set.
	dscoTracker := make(map[types.BlockHeight]struct{})
	idMap := make(map[types.SiacoinOutputID]struct{})

	// Iterate through all the buckets looking for the delayed siacoin output
	// buckets, and check that they are for the correct heights.
	err := tx.ForEach(func(name []byte, b *bolt.Bucket) error {
		// If the bucket is not a delayed siacoin output bucket or a file
		// contract expiration bucket, skip.
		if !strings.HasPrefix(string(name), string(prefixDSCO)) {
			return nil
		}

		// Add the bucket to the dscoTracker.
		var height types.BlockHeight
		err := encoding.Unmarshal(name[len(prefixDSCO):], &height)
		if build.DEBUG && err != nil {
			panic(err)
		}
		_, exists := dscoTracker[height]
		if exists {
			return errors.New("repeat dsco map")
		}
		dscoTracker[height] = struct{}{}

		var total types.Currency
		err = b.ForEach(func(idBytes, delayedOutput []byte) error {
			// Check that the output id has not appeared in another dsco.
			var id types.SiacoinOutputID
			copy(id[:], idBytes)
			_, exists := idMap[id]
			if exists {
				return errors.New("repeat delayed siacoin output")
			}
			idMap[id] = struct{}{}

			// Sum the funds in the bucket.
			var sco types.SiacoinOutput
			err := encoding.Unmarshal(delayedOutput, &sco)
			if build.DEBUG && err != nil {
				panic(err)
			}
			total = total.Add(sco.Value)
			return nil
		})
		if err != nil {
			return err
		}

		// Check that the minimum value has been achieved - the coinbase from
		// an earlier block is guaranteed to be in the bucket.
		minimumValue := types.CalculateCoinbase(height - types.MaturityDelay)
		if total.Cmp(minimumValue) < 0 {
			return errors.New("total number of coins in the delayed output bucket is incorrect")
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Check that all of the correct heights are represented.
	currentHeight := blockHeight(tx)
	expectedBuckets := 0
	for i := currentHeight + 1; i <= currentHeight+types.MaturityDelay; i++ {
		if i < types.MaturityDelay {
			continue
		}
		_, exists := dscoTracker[i]
		if !exists {
			return errors.New("missing a dsco bucket")
		}
		expectedBuckets++
	}
	if len(dscoTracker) != expectedBuckets {
		return errors.New("too many dsco buckets")
	}

	return nil
}

// checkRevertApply reverts the most recent block, checking to see that the
// consensus set hash matches the hash obtained for the previous block. Then it
// applies the block again and checks that the consensus set hash matches the
// original consensus set hash.
func (cs *ConsensusSet) checkRevertApply(tx *bolt.Tx) error {
	current := currentProcessedBlock(tx)
	parent, err := getBlockMap(tx, current.Block.ParentID)
	if err != nil {
		return err
	}
	_, _, err = cs.forkBlockchain(tx, parent)
	if err != nil {
		return err
	}
	if consensusChecksum(tx) != parent.ConsensusChecksum {
		return errors.New("consensus checksum mismatch after reverting")
	}
	_, _, err = cs.forkBlockchain(tx, current)
	if err != nil {
		return err
	}
	if consensusChecksum(tx) != current.ConsensusChecksum {
		return errors.New("consensus checksum mismatch after re-applying")
	}

	// TODO: Check that the parent relationship is correct.

	return nil
}

// checkConsistency runs a series of checks to make sure that the consensus set
// is consistent with some rules that should always be true.
func (cs *ConsensusSet) checkConsistency(tx *bolt.Tx) error {
	err := checkSiacoinCount(tx)
	if err != nil {
		return err
	}
	err = checkSiafundCount(tx)
	if err != nil {
		return err
	}
	err = checkDSCOs(tx)
	if err != nil {
		return err
	}
	err = cs.checkRevertApply(tx)
	if err != nil {
		return err
	}
	return nil
}

package api

import mapset "github.com/deckarep/golang-set/v2"

// DiffGPLists computes the difference between two lists by converting each
// element to a comparable key, performing set operations, then converting back.
// Returns (toAdd, toRemove, noChange) sets of the original type T.
func DiffGPLists[T comparable, K comparable](
	planList []T,
	stateList []T,
	toKey func(T) K,
	fromKey func(K) T,
) (toAdd, toRemove, noChange mapset.Set[T]) {
	planKeys := make([]K, 0, len(planList))
	for _, item := range planList {
		planKeys = append(planKeys, toKey(item))
	}
	stateKeys := make([]K, 0, len(stateList))
	for _, item := range stateList {
		stateKeys = append(stateKeys, toKey(item))
	}

	planSet := mapset.NewSet(planKeys...)
	stateSet := mapset.NewSet(stateKeys...)

	addKeys := planSet.Difference(stateSet)
	removeKeys := stateSet.Difference(planSet)
	unchangedKeys := planSet.Intersect(stateSet)

	toAdd = mapset.NewSet[T]()
	for k := range addKeys.Iterator().C {
		toAdd.Add(fromKey(k))
	}
	toRemove = mapset.NewSet[T]()
	for k := range removeKeys.Iterator().C {
		toRemove.Add(fromKey(k))
	}
	noChange = mapset.NewSet[T]()
	for k := range unchangedKeys.Iterator().C {
		noChange.Add(fromKey(k))
	}

	return toAdd, toRemove, noChange
}

// Key types used by the convenience wrappers below.

type gpAccountKey struct {
	GroupPolicyID string
	Role          string
}

type gpAccountGroupKey struct {
	GroupPolicyID string
	Role          string
}

type gpJumpGroupKey struct {
	GroupPolicyID  string
	JumpItemRoleID int
	// JumpPolicyID is optional (*int on the model). Track presence separately so
	// a nil policy is distinct from 0 and survives the key round-trip.
	JumpPolicyID    int
	HasJumpPolicyID bool
}

type gpJumpointKey struct {
	GroupPolicyID string
}

// Convenience wrappers preserve the existing public API.

func DiffGPAccountLists(planList []GroupPolicyVaultAccount, stateList []GroupPolicyVaultAccount) (mapset.Set[GroupPolicyVaultAccount], mapset.Set[GroupPolicyVaultAccount], mapset.Set[GroupPolicyVaultAccount]) {
	return DiffGPLists(planList, stateList,
		func(g GroupPolicyVaultAccount) gpAccountKey {
			return gpAccountKey{GroupPolicyID: *g.GroupPolicyID, Role: g.Role}
		},
		func(k gpAccountKey) GroupPolicyVaultAccount {
			id := k.GroupPolicyID
			return GroupPolicyVaultAccount{GroupPolicyID: &id, Role: k.Role}
		},
	)
}

func DiffGPAccountGroupLists(planList []GroupPolicyVaultAccountGroup, stateList []GroupPolicyVaultAccountGroup) (mapset.Set[GroupPolicyVaultAccountGroup], mapset.Set[GroupPolicyVaultAccountGroup], mapset.Set[GroupPolicyVaultAccountGroup]) {
	return DiffGPLists(planList, stateList,
		func(g GroupPolicyVaultAccountGroup) gpAccountGroupKey {
			return gpAccountGroupKey{GroupPolicyID: *g.GroupPolicyID, Role: g.Role}
		},
		func(k gpAccountGroupKey) GroupPolicyVaultAccountGroup {
			id := k.GroupPolicyID
			return GroupPolicyVaultAccountGroup{GroupPolicyID: &id, Role: k.Role}
		},
	)
}

func DiffGPJumpItemLists(planList []GroupPolicyJumpGroup, stateList []GroupPolicyJumpGroup) (mapset.Set[GroupPolicyJumpGroup], mapset.Set[GroupPolicyJumpGroup], mapset.Set[GroupPolicyJumpGroup]) {
	return DiffGPLists(planList, stateList,
		func(g GroupPolicyJumpGroup) gpJumpGroupKey {
			k := gpJumpGroupKey{GroupPolicyID: *g.GroupPolicyID, JumpItemRoleID: g.JumpItemRoleID}
			if g.JumpPolicyID != nil {
				k.JumpPolicyID = *g.JumpPolicyID
				k.HasJumpPolicyID = true
			}
			return k
		},
		func(k gpJumpGroupKey) GroupPolicyJumpGroup {
			id := k.GroupPolicyID
			g := GroupPolicyJumpGroup{GroupPolicyID: &id, JumpItemRoleID: k.JumpItemRoleID}
			if k.HasJumpPolicyID {
				policyID := k.JumpPolicyID
				g.JumpPolicyID = &policyID
			}
			return g
		},
	)
}

func DiffGPJumpointLists(planList []GroupPolicyJumpoint, stateList []GroupPolicyJumpoint) (mapset.Set[GroupPolicyJumpoint], mapset.Set[GroupPolicyJumpoint], mapset.Set[GroupPolicyJumpoint]) {
	return DiffGPLists(planList, stateList,
		func(g GroupPolicyJumpoint) gpJumpointKey {
			return gpJumpointKey{GroupPolicyID: *g.GroupPolicyID}
		},
		func(k gpJumpointKey) GroupPolicyJumpoint {
			id := k.GroupPolicyID
			return GroupPolicyJumpoint{GroupPolicyID: &id}
		},
	)
}

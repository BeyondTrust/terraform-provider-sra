package api

import mapset "github.com/deckarep/golang-set/v2"

// keySet converts a list to a set of comparable keys.
func keySet[T any, K comparable](items []T, toKey func(T) K) mapset.Set[K] {
	s := mapset.NewSet[K]()
	for _, item := range items {
		s.Add(toKey(item))
	}
	return s
}

// mapSet converts a set of keys back to a set of the original type.
func mapSet[K comparable, T comparable](keys mapset.Set[K], fromKey func(K) T) mapset.Set[T] {
	out := mapset.NewSet[T]()
	for _, k := range keys.ToSlice() {
		out.Add(fromKey(k))
	}
	return out
}

// DiffGPLists computes the difference between two lists by converting each
// element to a comparable key, performing set operations, then converting back.
// Returns (toAdd, toRemove, noChange) sets of the original type T.
func DiffGPLists[T comparable, K comparable](
	planList []T,
	stateList []T,
	toKey func(T) K,
	fromKey func(K) T,
) (toAdd, toRemove, noChange mapset.Set[T]) {
	planSet := keySet(planList, toKey)
	stateSet := keySet(stateList, toKey)

	return mapSet(planSet.Difference(stateSet), fromKey),
		mapSet(stateSet.Difference(planSet), fromKey),
		mapSet(planSet.Intersect(stateSet), fromKey)
}

// Key types used by the convenience wrappers below.

// gpRoleKey is shared by the account and account-group membership types,
// which are keyed identically (group policy ID + role).
type gpRoleKey struct {
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
		func(g GroupPolicyVaultAccount) gpRoleKey {
			return gpRoleKey{GroupPolicyID: *g.GroupPolicyID, Role: g.Role}
		},
		func(k gpRoleKey) GroupPolicyVaultAccount {
			id := k.GroupPolicyID
			return GroupPolicyVaultAccount{GroupPolicyID: &id, Role: k.Role}
		},
	)
}

func DiffGPAccountGroupLists(planList []GroupPolicyVaultAccountGroup, stateList []GroupPolicyVaultAccountGroup) (mapset.Set[GroupPolicyVaultAccountGroup], mapset.Set[GroupPolicyVaultAccountGroup], mapset.Set[GroupPolicyVaultAccountGroup]) {
	return DiffGPLists(planList, stateList,
		func(g GroupPolicyVaultAccountGroup) gpRoleKey {
			return gpRoleKey{GroupPolicyID: *g.GroupPolicyID, Role: g.Role}
		},
		func(k gpRoleKey) GroupPolicyVaultAccountGroup {
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

package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiffGPAccountLists(t *testing.T) {
	t.Parallel()

	gpID1 := "1"
	accountID1 := 1
	toAddItem := GroupPolicyVaultAccount{
		GroupPolicyID: &gpID1,
		AccountID:     &accountID1,
		Role:          "role1",
	}

	gpID2 := "2"
	accountID2 := 2
	toRemoveItem := GroupPolicyVaultAccount{
		GroupPolicyID: &gpID2,
		AccountID:     &accountID2,
		Role:          "role2",
	}

	gpID3 := "3"
	accountID3 := 3
	noChangeItem := GroupPolicyVaultAccount{
		GroupPolicyID: &gpID3,
		AccountID:     &accountID3,
		Role:          "role3",
	}

	set1 := []GroupPolicyVaultAccount{toAddItem, noChangeItem}
	set2 := []GroupPolicyVaultAccount{toRemoveItem, noChangeItem}

	toAdd, toRemove, noChange := DiffGPAccountLists(set1, set2)

	assert.Len(t, toAdd.ToSlice(), 1)
	assert.Len(t, toRemove.ToSlice(), 1)
	assert.Len(t, noChange.ToSlice(), 1)

	assert.Equal(t, *toAddItem.GroupPolicyID, *toAdd.ToSlice()[0].GroupPolicyID)
	assert.Nil(t, toAdd.ToSlice()[0].AccountID)
	assert.Equal(t, toAddItem.Role, toAdd.ToSlice()[0].Role)

	assert.Equal(t, *toRemoveItem.GroupPolicyID, *toRemove.ToSlice()[0].GroupPolicyID)
	assert.Nil(t, toRemove.ToSlice()[0].AccountID)
	assert.Equal(t, toRemoveItem.Role, toRemove.ToSlice()[0].Role)

	assert.Equal(t, *noChangeItem.GroupPolicyID, *noChange.ToSlice()[0].GroupPolicyID)
	assert.Nil(t, noChange.ToSlice()[0].AccountID)
	assert.Equal(t, noChangeItem.Role, noChange.ToSlice()[0].Role)
}

func TestDiffGPAccountGroupLists(t *testing.T) {
	t.Parallel()

	gpID1 := "1"
	accountID1 := 1
	toAddItem := GroupPolicyVaultAccountGroup{
		GroupPolicyID:  &gpID1,
		AccountGroupID: &accountID1,
		Role:           "role1",
	}

	gpID2 := "2"
	accountID2 := 2
	toRemoveItem := GroupPolicyVaultAccountGroup{
		GroupPolicyID:  &gpID2,
		AccountGroupID: &accountID2,
		Role:           "role2",
	}

	gpID3 := "3"
	accountID3 := 3
	noChangeItem := GroupPolicyVaultAccountGroup{
		GroupPolicyID:  &gpID3,
		AccountGroupID: &accountID3,
		Role:           "role3",
	}

	set1 := []GroupPolicyVaultAccountGroup{toAddItem, noChangeItem}
	set2 := []GroupPolicyVaultAccountGroup{toRemoveItem, noChangeItem}

	toAdd, toRemove, noChange := DiffGPAccountGroupLists(set1, set2)

	assert.Len(t, toAdd.ToSlice(), 1)
	assert.Len(t, toRemove.ToSlice(), 1)
	assert.Len(t, noChange.ToSlice(), 1)

	assert.Equal(t, *toAddItem.GroupPolicyID, *toAdd.ToSlice()[0].GroupPolicyID)
	assert.Nil(t, toAdd.ToSlice()[0].AccountGroupID)
	assert.Equal(t, toAddItem.Role, toAdd.ToSlice()[0].Role)

	assert.Equal(t, *toRemoveItem.GroupPolicyID, *toRemove.ToSlice()[0].GroupPolicyID)
	assert.Nil(t, toRemove.ToSlice()[0].AccountGroupID)
	assert.Equal(t, toRemoveItem.Role, toRemove.ToSlice()[0].Role)

	assert.Equal(t, *noChangeItem.GroupPolicyID, *noChange.ToSlice()[0].GroupPolicyID)
	assert.Nil(t, noChange.ToSlice()[0].AccountGroupID)
	assert.Equal(t, noChangeItem.Role, noChange.ToSlice()[0].Role)
}

func TestDiffGPJumpItemLists(t *testing.T) {
	t.Parallel()

	gpID1 := "1"
	groupID1 := 1
	policyID1 := 1
	toAddItem := GroupPolicyJumpGroup{
		GroupPolicyID:  &gpID1,
		JumpGroupID:    &groupID1,
		JumpItemRoleID: 1,
		JumpPolicyID:   &policyID1,
	}

	gpID2 := "2"
	groupID2 := 2
	policyID2 := 2
	toRemoveItem := GroupPolicyJumpGroup{
		GroupPolicyID:  &gpID2,
		JumpGroupID:    &groupID2,
		JumpItemRoleID: 2,
		JumpPolicyID:   &policyID2,
	}

	gpID3 := "3"
	groupID3 := 3
	policyID3 := 3
	noChangeItem := GroupPolicyJumpGroup{
		GroupPolicyID:  &gpID3,
		JumpGroupID:    &groupID3,
		JumpItemRoleID: 3,
		JumpPolicyID:   &policyID3,
	}

	set1 := []GroupPolicyJumpGroup{toAddItem, noChangeItem}
	set2 := []GroupPolicyJumpGroup{toRemoveItem, noChangeItem}

	toAdd, toRemove, noChange := DiffGPJumpItemLists(set1, set2)

	assert.Len(t, toAdd.ToSlice(), 1)
	assert.Len(t, toRemove.ToSlice(), 1)
	assert.Len(t, noChange.ToSlice(), 1)

	assert.Equal(t, *toAddItem.GroupPolicyID, *toAdd.ToSlice()[0].GroupPolicyID)
	assert.Nil(t, toAdd.ToSlice()[0].JumpGroupID)
	assert.Equal(t, toAddItem.JumpItemRoleID, toAdd.ToSlice()[0].JumpItemRoleID)
	assert.Equal(t, *toAddItem.JumpPolicyID, *toAdd.ToSlice()[0].JumpPolicyID)

	assert.Equal(t, *toRemoveItem.GroupPolicyID, *toRemove.ToSlice()[0].GroupPolicyID)
	assert.Nil(t, toRemove.ToSlice()[0].JumpGroupID)
	assert.Equal(t, toRemoveItem.JumpItemRoleID, toRemove.ToSlice()[0].JumpItemRoleID)
	assert.Equal(t, *toRemoveItem.JumpPolicyID, *toRemove.ToSlice()[0].JumpPolicyID)

	assert.Equal(t, *noChangeItem.GroupPolicyID, *noChange.ToSlice()[0].GroupPolicyID)
	assert.Nil(t, noChange.ToSlice()[0].JumpGroupID)
	assert.Equal(t, noChangeItem.JumpItemRoleID, noChange.ToSlice()[0].JumpItemRoleID)
	assert.Equal(t, *noChangeItem.JumpPolicyID, *noChange.ToSlice()[0].JumpPolicyID)
}

func TestDiffGPJumpointLists(t *testing.T) {
	t.Parallel()

	gpID1 := "1"
	jumpointID1 := 1
	toAddItem := GroupPolicyJumpoint{
		GroupPolicyID: &gpID1,
		JumpointID:    &jumpointID1,
	}

	gpID2 := "2"
	jumpointID2 := 2
	toRemoveItem := GroupPolicyJumpoint{
		GroupPolicyID: &gpID2,
		JumpointID:    &jumpointID2,
	}

	gpID3 := "3"
	jumpointID3 := 3
	noChangeItem := GroupPolicyJumpoint{
		GroupPolicyID: &gpID3,
		JumpointID:    &jumpointID3,
	}

	set1 := []GroupPolicyJumpoint{toAddItem, noChangeItem}
	set2 := []GroupPolicyJumpoint{toRemoveItem, noChangeItem}

	toAdd, toRemove, noChange := DiffGPJumpointLists(set1, set2)

	assert.Len(t, toAdd.ToSlice(), 1)
	assert.Len(t, toRemove.ToSlice(), 1)
	assert.Len(t, noChange.ToSlice(), 1)

	assert.Equal(t, *toAddItem.GroupPolicyID, *toAdd.ToSlice()[0].GroupPolicyID)
	assert.Equal(t, *toRemoveItem.GroupPolicyID, *toRemove.ToSlice()[0].GroupPolicyID)
	assert.Equal(t, *noChangeItem.GroupPolicyID, *noChange.ToSlice()[0].GroupPolicyID)
}

package api

import mapset "github.com/deckarep/golang-set/v2"

type noPointerGPAccount struct {
	GroupPolicyID string
	Role          string
}

func DiffGPAccountLists(planList []GroupPolicyVaultAccount, stateList []GroupPolicyVaultAccount) (mapset.Set[GroupPolicyVaultAccount], mapset.Set[GroupPolicyVaultAccount], mapset.Set[GroupPolicyVaultAccount]) {
	newPlanList := []noPointerGPAccount{}
	for _, i := range planList {
		newPlanList = append(newPlanList, noPointerGPAccount{
			GroupPolicyID: *i.GroupPolicyID,
			Role:          i.Role,
		})
	}
	newSetList := []noPointerGPAccount{}
	for _, i := range stateList {
		newSetList = append(newSetList, noPointerGPAccount{
			GroupPolicyID: *i.GroupPolicyID,
			Role:          i.Role,
		})
	}

	setGPList := mapset.NewSet(newPlanList...)
	setGPStateList := mapset.NewSet(newSetList...)

	toAdd := setGPList.Difference(setGPStateList)
	toRemove := setGPStateList.Difference(setGPList)
	noChange := setGPList.Intersect(setGPStateList)

	toAddReturn := mapset.NewSet[GroupPolicyVaultAccount]()
	for i := range toAdd.Iterator().C {
		toAddReturn.Add(GroupPolicyVaultAccount{
			GroupPolicyID: &i.GroupPolicyID,
			Role:          i.Role,
		})
	}
	toRemoveReturn := mapset.NewSet[GroupPolicyVaultAccount]()
	for i := range toRemove.Iterator().C {
		toRemoveReturn.Add(GroupPolicyVaultAccount{
			GroupPolicyID: &i.GroupPolicyID,
			Role:          i.Role,
		})
	}
	noChangeReturn := mapset.NewSet[GroupPolicyVaultAccount]()
	for i := range noChange.Iterator().C {
		noChangeReturn.Add(GroupPolicyVaultAccount{
			GroupPolicyID: &i.GroupPolicyID,
			Role:          i.Role,
		})
	}

	return toAddReturn, toRemoveReturn, noChangeReturn
}

type noPointerGPAccountGroup struct {
	GroupPolicyID string
	Role          string
}

func DiffGPAccountGroupLists(planList []GroupPolicyVaultAccountGroup, stateList []GroupPolicyVaultAccountGroup) (mapset.Set[GroupPolicyVaultAccountGroup], mapset.Set[GroupPolicyVaultAccountGroup], mapset.Set[GroupPolicyVaultAccountGroup]) {
	newPlanList := []noPointerGPAccountGroup{}
	for _, i := range planList {
		newPlanList = append(newPlanList, noPointerGPAccountGroup{
			GroupPolicyID: *i.GroupPolicyID,
			Role:          i.Role,
		})
	}
	newSetList := []noPointerGPAccountGroup{}
	for _, i := range stateList {
		newSetList = append(newSetList, noPointerGPAccountGroup{
			GroupPolicyID: *i.GroupPolicyID,
			Role:          i.Role,
		})
	}

	setGPList := mapset.NewSet(newPlanList...)
	setGPStateList := mapset.NewSet(newSetList...)

	toAdd := setGPList.Difference(setGPStateList)
	toRemove := setGPStateList.Difference(setGPList)
	noChange := setGPList.Intersect(setGPStateList)

	toAddReturn := mapset.NewSet[GroupPolicyVaultAccountGroup]()
	for i := range toAdd.Iterator().C {
		toAddReturn.Add(GroupPolicyVaultAccountGroup{
			GroupPolicyID: &i.GroupPolicyID,
			Role:          i.Role,
		})
	}
	toRemoveReturn := mapset.NewSet[GroupPolicyVaultAccountGroup]()
	for i := range toRemove.Iterator().C {
		toRemoveReturn.Add(GroupPolicyVaultAccountGroup{
			GroupPolicyID: &i.GroupPolicyID,
			Role:          i.Role,
		})
	}
	noChangeReturn := mapset.NewSet[GroupPolicyVaultAccountGroup]()
	for i := range noChange.Iterator().C {
		noChangeReturn.Add(GroupPolicyVaultAccountGroup{
			GroupPolicyID: &i.GroupPolicyID,
			Role:          i.Role,
		})
	}

	return toAddReturn, toRemoveReturn, noChangeReturn
}

type noPointerGPJumpGroup struct {
	GroupPolicyID  string
	JumpItemRoleID int
	JumpPolicyID   int
}

func DiffGPJumpItemLists(planList []GroupPolicyJumpGroup, stateList []GroupPolicyJumpGroup) (mapset.Set[GroupPolicyJumpGroup], mapset.Set[GroupPolicyJumpGroup], mapset.Set[GroupPolicyJumpGroup]) {
	newPlanList := []noPointerGPJumpGroup{}
	for _, i := range planList {
		newPlanList = append(newPlanList, noPointerGPJumpGroup{
			GroupPolicyID:  *i.GroupPolicyID,
			JumpItemRoleID: i.JumpItemRoleID,
			JumpPolicyID:   i.JumpPolicyID,
		})
	}
	newSetList := []noPointerGPJumpGroup{}
	for _, i := range stateList {
		newSetList = append(newSetList, noPointerGPJumpGroup{
			GroupPolicyID:  *i.GroupPolicyID,
			JumpItemRoleID: i.JumpItemRoleID,
			JumpPolicyID:   i.JumpPolicyID,
		})
	}

	setGPList := mapset.NewSet(newPlanList...)
	setGPStateList := mapset.NewSet(newSetList...)

	toAdd := setGPList.Difference(setGPStateList)
	toRemove := setGPStateList.Difference(setGPList)
	noChange := setGPList.Intersect(setGPStateList)

	toAddReturn := mapset.NewSet[GroupPolicyJumpGroup]()
	for i := range toAdd.Iterator().C {
		toAddReturn.Add(GroupPolicyJumpGroup{
			GroupPolicyID:  &i.GroupPolicyID,
			JumpItemRoleID: i.JumpItemRoleID,
			JumpPolicyID:   i.JumpPolicyID,
		})
	}
	toRemoveReturn := mapset.NewSet[GroupPolicyJumpGroup]()
	for i := range toRemove.Iterator().C {
		toRemoveReturn.Add(GroupPolicyJumpGroup{
			GroupPolicyID:  &i.GroupPolicyID,
			JumpItemRoleID: i.JumpItemRoleID,
			JumpPolicyID:   i.JumpPolicyID,
		})
	}
	noChangeReturn := mapset.NewSet[GroupPolicyJumpGroup]()
	for i := range noChange.Iterator().C {
		noChangeReturn.Add(GroupPolicyJumpGroup{
			GroupPolicyID:  &i.GroupPolicyID,
			JumpItemRoleID: i.JumpItemRoleID,
			JumpPolicyID:   i.JumpPolicyID,
		})
	}

	return toAddReturn, toRemoveReturn, noChangeReturn
}

type noPointerGPJumpoint struct {
	GroupPolicyID string
}

func DiffGPJumpointLists(planList []GroupPolicyJumpoint, stateList []GroupPolicyJumpoint) (mapset.Set[GroupPolicyJumpoint], mapset.Set[GroupPolicyJumpoint], mapset.Set[GroupPolicyJumpoint]) {
	newPlanList := []noPointerGPJumpoint{}
	for _, i := range planList {
		newPlanList = append(newPlanList, noPointerGPJumpoint{
			GroupPolicyID: *i.GroupPolicyID,
		})
	}
	newSetList := []noPointerGPJumpoint{}
	for _, i := range stateList {
		newSetList = append(newSetList, noPointerGPJumpoint{
			GroupPolicyID: *i.GroupPolicyID,
		})
	}

	setGPList := mapset.NewSet(newPlanList...)
	setGPStateList := mapset.NewSet(newSetList...)

	toAdd := setGPList.Difference(setGPStateList)
	toRemove := setGPStateList.Difference(setGPList)
	noChange := setGPList.Intersect(setGPStateList)

	toAddReturn := mapset.NewSet[GroupPolicyJumpoint]()
	for i := range toAdd.Iterator().C {
		toAddReturn.Add(GroupPolicyJumpoint{
			GroupPolicyID: &i.GroupPolicyID,
		})
	}
	toRemoveReturn := mapset.NewSet[GroupPolicyJumpoint]()
	for i := range toRemove.Iterator().C {
		toRemoveReturn.Add(GroupPolicyJumpoint{
			GroupPolicyID: &i.GroupPolicyID,
		})
	}
	noChangeReturn := mapset.NewSet[GroupPolicyJumpoint]()
	for i := range noChange.Iterator().C {
		noChangeReturn.Add(GroupPolicyJumpoint{
			GroupPolicyID: &i.GroupPolicyID,
		})
	}

	return toAddReturn, toRemoveReturn, noChangeReturn
}

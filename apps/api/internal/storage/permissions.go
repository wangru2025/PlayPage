package storage

import "ai-static-host/api/internal/domain"

func normalizeStoredPermissions(permissions domain.PermissionSet) domain.PermissionSet {
	if !permissions.PublicWrite && (permissions.AuthWrite || permissions.OwnerWrite || permissions.OwnerDelete) {
		permissions.PublicWrite = true
	}
	permissions.AuthWrite = false
	permissions.OwnerWrite = false
	permissions.OwnerDelete = false
	return permissions
}

func normalizeCollectionList(items []domain.Collection) []domain.Collection {
	next := make([]domain.Collection, len(items))
	copy(next, items)
	for index := range next {
		next[index].Permissions = normalizeStoredPermissions(next[index].Permissions)
	}
	return next
}

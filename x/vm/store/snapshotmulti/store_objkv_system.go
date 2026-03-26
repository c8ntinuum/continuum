//go:build system_test

package snapshotmulti

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"
)

// GetObjKVStore returns an underlying ObjKVStore by key.
//
// This is compiled only for system tests, where newer store interfaces require
// ObjKVStore access on CacheMultiStore/MultiStore.
func (s *Store) GetObjKVStore(key storetypes.StoreKey) storetypes.ObjKVStore {
	store := s.stores[key]
	if key == nil || store == nil {
		panic(fmt.Sprintf("kv store with key %v has not been registered in stores", key))
	}

	objStore, ok := any(store.CurrentStore()).(storetypes.ObjKVStore)
	if !ok {
		panic(fmt.Sprintf("store with key %v is not ObjKVStore", key))
	}

	return objStore
}

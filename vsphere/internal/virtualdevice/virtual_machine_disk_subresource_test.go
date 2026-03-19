// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package virtualdevice

import (
	"testing"

	"github.com/vmware/govmomi/vim25/types"
)

// newVirtualDiskWithBacking creates a VirtualDisk with the given backing info set.
func newVirtualDiskWithBacking(backing types.BaseVirtualDeviceBackingInfo) *types.VirtualDisk {
	disk := &types.VirtualDisk{}
	disk.Backing = backing
	return disk
}

func TestDiskUUIDMatch_RDM(t *testing.T) {
	uuid := "6000C29e-1234-5678-9abc-def012345678"
	cases := []struct {
		name     string
		device   types.BaseVirtualDevice
		uuid     string
		expected bool
	}{
		{
			name: "RDM backing - matching UUID",
			device: newVirtualDiskWithBacking(&types.VirtualDiskRawDiskMappingVer1BackingInfo{
				Uuid: uuid,
			}),
			uuid:     uuid,
			expected: true,
		},
		{
			name: "RDM backing - non-matching UUID",
			device: newVirtualDiskWithBacking(&types.VirtualDiskRawDiskMappingVer1BackingInfo{
				Uuid: uuid,
			}),
			uuid:     "different-uuid",
			expected: false,
		},
		{
			name: "flat backing - matching UUID",
			device: newVirtualDiskWithBacking(&types.VirtualDiskFlatVer2BackingInfo{
				Uuid: uuid,
			}),
			uuid:     uuid,
			expected: true,
		},
		{
			name: "sparse backing - matching UUID",
			device: newVirtualDiskWithBacking(&types.VirtualDiskSparseVer2BackingInfo{
				Uuid: uuid,
			}),
			uuid:     uuid,
			expected: true,
		},
		{
			name:     "non-disk device",
			device:   &types.VirtualCdrom{},
			uuid:     uuid,
			expected: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := diskUUIDMatch(tc.device, tc.uuid)
			if tc.expected != actual {
				t.Fatalf("expected %v, got %v", tc.expected, actual)
			}
		})
	}
}

func TestVirtualDiskToSchemaPropsMap_RDM(t *testing.T) {
	dsRef := types.ManagedObjectReference{Type: "Datastore", Value: "datastore-1"}

	cases := []struct {
		name        string
		disk        *types.VirtualDisk
		checkFields map[string]interface{}
	}{
		{
			name: "RDM disk returns RDM properties",
			disk: newVirtualDiskWithBacking(&types.VirtualDiskRawDiskMappingVer1BackingInfo{
				VirtualDeviceFileBackingInfo: types.VirtualDeviceFileBackingInfo{
					Datastore: &dsRef,
				},
				DiskMode:          "persistent",
				DeviceName:        "/vmfs/devices/disks/naa.123",
				CompatibilityMode: "virtualMode",
			}),
			checkFields: map[string]interface{}{
				"datastore_id":           "datastore-1",
				"disk_mode":              "persistent",
				"rdm_device_name":        "/vmfs/devices/disks/naa.123",
				"rdm_compatibility_mode": "virtualMode",
			},
		},
		{
			name: "RDM disk with nil datastore",
			disk: newVirtualDiskWithBacking(&types.VirtualDiskRawDiskMappingVer1BackingInfo{
				DiskMode:          "persistent",
				DeviceName:        "/vmfs/devices/disks/naa.456",
				CompatibilityMode: "physicalMode",
			}),
			checkFields: map[string]interface{}{
				"disk_mode":              "persistent",
				"rdm_device_name":        "/vmfs/devices/disks/naa.456",
				"rdm_compatibility_mode": "physicalMode",
			},
		},
		{
			name: "flat VMDK disk returns flat properties",
			disk: newVirtualDiskWithBacking(&types.VirtualDiskFlatVer2BackingInfo{
				VirtualDeviceFileBackingInfo: types.VirtualDeviceFileBackingInfo{
					Datastore: &dsRef,
				},
				DiskMode:     "persistent",
				WriteThrough: func() *bool { b := false; return &b }(),
			}),
			checkFields: map[string]interface{}{
				"datastore_id": "datastore-1",
				"disk_mode":    "persistent",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := virtualDiskToSchemaPropsMap(tc.disk)
			for key, expected := range tc.checkFields {
				actual, ok := result[key]
				if !ok {
					t.Fatalf("expected key %q in result, but it was missing", key)
				}
				if actual != expected {
					t.Fatalf("key %q: expected %v, got %v", key, expected, actual)
				}
			}
		})
	}
}

func TestVirtualDiskToSchemaPropsMap_RDM_NoFlatFields(t *testing.T) {
	// Verify that RDM disks do NOT have flat-specific fields
	disk := newVirtualDiskWithBacking(&types.VirtualDiskRawDiskMappingVer1BackingInfo{
		DeviceName:        "/vmfs/devices/disks/naa.789",
		CompatibilityMode: "virtualMode",
		DiskMode:          "persistent",
	})
	result := virtualDiskToSchemaPropsMap(disk)

	flatOnlyFields := []string{"eagerly_scrub", "thin_provisioned", "write_through", "disk_sharing"}
	for _, key := range flatOnlyFields {
		if _, ok := result[key]; ok {
			t.Fatalf("RDM disk should not have flat-specific field %q", key)
		}
	}
}

func TestDiskCapacityInGiB(t *testing.T) {
	cases := []struct {
		name     string
		subject  *types.VirtualDisk
		expected int
	}{
		{
			name: "capacityInBytes - integer GiB",
			subject: &types.VirtualDisk{
				CapacityInBytes: 4294967296,
				CapacityInKB:    4194304,
			},
			expected: 4,
		},
		{
			name: "capacityInKB - integer GiB",
			subject: &types.VirtualDisk{
				CapacityInKB: 4194304,
			},
			expected: 4,
		},
		{
			name: "capacityInBytes - non-integer GiB",
			subject: &types.VirtualDisk{
				CapacityInBytes: 4294968320,
				CapacityInKB:    4194305,
			},
			expected: 5,
		},
		{
			name: "capacityInKB - non-integer GiB",
			subject: &types.VirtualDisk{
				CapacityInKB: 4194305,
			},
			expected: 5,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := diskCapacityInGiB(tc.subject)
			if tc.expected != actual {
				t.Fatalf("expected %d, got %d", tc.expected, actual)
			}
		})
	}
}

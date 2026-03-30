# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Terraform provider for VMware vSphere, built with the Terraform Plugin SDK v2 and the govmomi library. Manages 45+ vSphere resources (VMs, clusters, datastores, networking, tags, etc.) and 32+ data sources.

## Build & Test Commands

```bash
make build          # Build provider (runs fmtcheck first)
make test           # Unit tests (30s timeout, 4 parallel)
make testacc        # Acceptance tests (requires TF_ACC=1, vSphere environment)
make fmt            # Format Go files
make fmtcheck       # Check Go formatting
make docs-check     # Validate documentation structure
make docs-hcl-lint  # Check HCL formatting in docs
make docs-hcl-fix   # Fix HCL formatting in docs
```

Run a single test:
```bash
go test ./vsphere/ -run TestAccResourceVSphereVirtualMachine_basic -v -timeout 360m
```

Run unit tests only (no vSphere required):
```bash
go test ./vsphere/ -run 'TestUnit' -v -timeout 30s
```

Acceptance tests require environment variables: `TF_ACC=1`, `VSPHERE_USER`, `VSPHERE_PASSWORD`, `VSPHERE_SERVER`, `VSPHERE_ALLOW_UNVERIFIED_SSL`.

## Linting

Uses golangci-lint v2 with: errcheck, gosec, govet, ineffassign, misspell, revive, staticcheck, unconvert, unused. Plus gofmt/goimports formatting. Config in `.golangci.yml`.

## Code Architecture

**Entry point:** `main.go` calls `vsphere.Provider()`.

**Provider package (`vsphere/`):** Contains all resource and data source implementations. Each resource follows the pattern:
- `resource_vsphere_<name>.go` - CRUD operations
- `resource_vsphere_<name>_test.go` - Acceptance tests
- `data_source_vsphere_<name>.go` / `data_source_vsphere_<name>_test.go` for data sources

**Provider registration:** `vsphere/provider.go` registers all resources and data sources, plus provider-level schema (connection config).

**Client configuration:** `vsphere/config.go` manages vSphere client creation with VIM and REST session persistence.

**Helper packages (`vsphere/internal/helper/`):** Domain-specific logic organized by vSphere object type (datacenter, datastore, hostsystem, virtualmachine, etc.). Resources delegate API interactions to these helpers.

**Virtual device subsystem (`vsphere/internal/virtualdevice/`):** Handles virtual machine hardware (disks, NICs, SCSI/SATA/NVMe controllers, CD-ROMs). This is the most complex part of the codebase due to device lifecycle management across create/read/update/delete.

**VM workflow (`vsphere/internal/vmworkflow/`):** Shared logic for VM creation workflows (clone, deploy from OVF, etc.).

**Structure files:** Complex resources have dedicated `*_structure.go` files containing schema builders for nested objects (e.g., `virtual_machine_config_structure.go`).

## Key Patterns

**Resource CRUD:** Each resource implements Create/Read/Update/Delete functions. Create and Update typically call Read at the end to refresh state. Most resources support import via `ResourceImporter`.

**Tags and custom attributes:** Most resources support vSphere tags (`vSphereTagAttributeKey: tagsSchema()`) and custom attributes (`customattribute.ConfigKey: customattribute.ConfigSchema()`). Use `processTagDiff` and `customattribute.GetDiffProcessorIfAttributesDefined` in CRUD operations.

**Client access:** `meta.(*Client).vimClient` for VIM API, `meta.(*Client).restClient` for REST API.

**Async operations:** Use `resource.StateChangeConf` with `Pending`/`Target` states and a `Refresh` function for polling.

## Commit Convention

Uses [Conventional Commits](https://conventionalcommits.org) with sign-off. Example: `feat: add support for x`.

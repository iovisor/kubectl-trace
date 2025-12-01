# Fix for NodeSelector kubernetes.io/hostname Reliability Issue

## Problem Description

The kubectl-trace tool was using the `kubernetes.io/hostname` label for node selection in trace jobs. This approach is unreliable because:

1. **AWS and other cloud providers**: NodeName might be fully qualified (e.g., `ip-10-0-1-123.ec2.internal`) while the `kubernetes.io/hostname` label contains only the short hostname (e.g., `ip-10-0-1-123`)
2. **Label inconsistencies**: The hostname label may not match the actual node name, causing trace jobs to fail scheduling
3. **Configuration drift**: The hostname label can be modified independently of the node name

## Solution Overview

The fix replaces the unreliable `kubernetes.io/hostname` label-based node selection with direct node name matching using `metadata.name` field selector. This ensures that trace jobs are always scheduled on the correct node regardless of hostname label configuration.

## Code Changes

### 1. Updated Job Creation (`pkg/tracejob/job.go`)

**Before:**
```go
MatchExpressions: []apiv1.NodeSelectorRequirement{
    apiv1.NodeSelectorRequirement{
        Key:      "kubernetes.io/hostname",
        Operator: apiv1.NodeSelectorOpIn,
        Values:   []string{nj.Target.Node},
    },
},
```

**After:**
```go
MatchFields: []apiv1.NodeSelectorRequirement{
    apiv1.NodeSelectorRequirement{
        Key:      "metadata.name",
        Operator: apiv1.NodeSelectorOpIn,
        Values:   []string{nj.Target.Node},
    },
},
```

### 2. Updated Node Target Resolution (`pkg/tracejob/selected_target.go`)

**Before:**
```go
labels := node.GetLabels()
val, ok := labels["kubernetes.io/hostname"]
if !ok {
    return nil, errors.NewErrorInvalid("label kubernetes.io/hostname not found in node")
}
target.Node = val
```

**After:**
```go
target.Node = node.Name
```

### 3. Enhanced Job Hostname Extraction (`pkg/tracejob/job.go`)

Updated the `jobHostname` function to:
- Prioritize the new `MatchFields` approach with `metadata.name`
- Maintain backward compatibility with existing `MatchExpressions` using `kubernetes.io/hostname`
- Provide clear error messages for debugging

## Test Coverage

### New Test Cases Added

1. **`TestJobNodeSelectorUsesNodeName`** - Verifies that created jobs use `MatchFields` with `metadata.name`
2. **`TestJobHostnameExtraction`** - Tests both new and old node selection methods for backward compatibility
3. **`TestResolveNodeTargetUsesNodeName`** - Ensures node resolution uses actual node name, not hostname label
4. **`TestResolvePodTargetUsesNodeName`** - Verifies pod-based targets use the correct node name from pod spec
5. **`TestResolveDeploymentTargetUsesNodeName`** - Tests deployment-based target resolution

### Test Scenarios Covered

- ✅ Fully qualified node names (AWS style: `ip-10-0-1-123.ec2.internal`)
- ✅ Short hostname labels (`ip-10-0-1-123`)
- ✅ Pod-based tracing with correct node name extraction
- ✅ Deployment-based tracing through pod resolution
- ✅ Backward compatibility with existing jobs

## Benefits

1. **Reliability**: Trace jobs will always schedule on the correct node
2. **Cloud Compatibility**: Works properly with AWS, GCP, Azure, and other cloud providers
3. **Backward Compatibility**: Existing trace jobs continue to function
4. **Simplicity**: Removes dependency on potentially inconsistent labels
5. **Performance**: Field-based selection is more efficient than label-based selection

## Migration Impact

- **Zero downtime**: Existing jobs continue to work
- **No configuration changes**: No updates required to cluster configuration
- **Transparent**: Users see no change in behavior
- **Future-proof**: Works with any Kubernetes cluster regardless of hostname labeling

## Verification

Run the test suite to verify the fix:

```bash
go test ./pkg/tracejob -v
```

All tests should pass, confirming that:
- New jobs use node name-based selection
- Existing jobs can still be parsed correctly
- Node resolution works for all target types (node, pod, deployment)

## Files Modified

- `pkg/tracejob/job.go` - Updated node affinity and hostname extraction
- `pkg/tracejob/selected_target.go` - Simplified node target resolution
- `pkg/tracejob/job_test.go` - Added comprehensive test coverage
- `pkg/tracejob/selected_target_test.go` - Added target resolution tests

## Backward Compatibility

The fix maintains full backward compatibility by:
- Supporting both `MatchFields` (new) and `MatchExpressions` (old) in job parsing
- Not breaking existing trace job functionality
- Allowing gradual migration of existing jobs

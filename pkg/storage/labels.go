package storage

import (
	"strconv"

	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
)

const maxLabelLength = 63

func newIndexLabels(owner, key string, rls *release.Release) map[string]string {
	lbls := map[string]string{
		"owner":   owner,
		"type":    "index",
		"name":    rls.Name,
		"version": strconv.Itoa(rls.Version),
		"status":  rls.Info.Status.String(),
	}

	// The key is something like sh.helm.release.v1.<rel.Name>.v<rel.Version>,
	// which can easily creep above the 63 character label value limit. For
	// backwards compatibility purposes, we'll add the key if it is under that
	// limit.
	if len(key) <= maxLabelLength {
		lbls["key"] = key
	}
	return lbls
}

func newChunkLabels(owner, key string, rls *release.Release) map[string]string {
	lbls := map[string]string{
		"owner":   owner,
		"type":    "chunk",
		"name":    rls.Name,
		"version": strconv.Itoa(rls.Version),
	}

	// The key is something like sh.helm.release.v1.<rel.Name>.v<rel.Version>,
	// which can easily creep above the 63 character label value limit. For
	// backwards compatibility purposes, we'll add the key if it is under that
	// limit.
	if len(key) <= maxLabelLength {
		lbls["key"] = key
	}
	return lbls
}

func newListIndicesLabelSelector(owner string) labels.Selector {
	return labels.Set{"owner": owner, "type": "index"}.AsSelector()
}

func newListAllForKeySelector(owner, key string, rls *release.Release) labels.Selector {
	// The key format is something like sh.helm.release.v1.<rel.Name>.v<rel.Version>,
	// which can easily creep above the 63 character label value limit. For
	// backwards compatibility purposes, we'll use the key if it is under that
	// limit.
	if len(key) <= maxLabelLength {
		return labels.Set{"owner": owner, "key": key}.AsSelector()
	}
	return labels.Set{"owner": owner, "name": rls.Name, "version": strconv.Itoa(rls.Version)}.AsSelector()
}

func newListChunksForKeySelector(owner, key string, rls *release.Release) labels.Selector {
	// The key is something like sh.helm.release.v1.<rel.Name>.v<rel.Version>,
	// which can easily creep above the 63 character label value limit. For
	// backwards compatibility purposes, we'll use the key if it is under that
	// limit.
	if len(key) <= maxLabelLength {
		return labels.Set{"owner": owner, "key": key, "type": "chunk"}.AsSelector()
	}
	return labels.Set{"owner": owner, "name": rls.Name, "version": strconv.Itoa(rls.Version), "type": "chunk"}.AsSelector()

}

var systemLabels = sets.New[string]("name", "owner", "status", "version", "key", "type", "createdAt", "modifiedAt")

// Checks if label is system
func isSystemLabel(key string) bool {
	return systemLabels.Has(key)
}

// Removes system labels from labels map
func filterSystemLabels(lbs map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range lbs {
		if !isSystemLabel(k) {
			result[k] = v
		}
	}
	return result
}

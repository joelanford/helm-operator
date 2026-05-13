package release

import (
	"fmt"

	helmrelease "helm.sh/helm/v4/pkg/release"
	release "helm.sh/helm/v4/pkg/release/v1"
)

func ToV1(rel helmrelease.Releaser) (*release.Release, error) {
	switch r := rel.(type) {
	case *release.Release:
		return r, nil
	case release.Release:
		return &r, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported release type: %T", rel)
	}
}

func SliceToV1(rels []helmrelease.Releaser) ([]*release.Release, error) {
	result := make([]*release.Release, 0, len(rels))
	for _, r := range rels {
		rel, err := ToV1(r)
		if err != nil {
			return nil, err
		}
		result = append(result, rel)
	}
	return result, nil
}

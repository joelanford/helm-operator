/*
Copyright 2020 The Operator-SDK Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package values

import (
	"context"

	"helm.sh/helm/v4/pkg/chart/common"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Mapper is an interface expected by the reconciler.WithValueMapper option.
//
// Deprecated: use Translator instead.
type Mapper interface {
	Map(common.Values) common.Values
}

type MapperFunc func(common.Values) common.Values

func (m MapperFunc) Map(v common.Values) common.Values {
	return m(v)
}

// Translator is an interface expected by the reconciler.WithValueTranslator option.
//
// Translate should return helm values based on the content of the unstructured object
// which is being reconciled.
//
// See also the option documentation.
type Translator interface {
	Translate(ctx context.Context, unstructured *unstructured.Unstructured) (common.Values, error)
}

// TranslatorFunc is a helper type for passing a function as a Translator.
type TranslatorFunc func(context.Context, *unstructured.Unstructured) (common.Values, error)

func (t TranslatorFunc) Translate(ctx context.Context, u *unstructured.Unstructured) (common.Values, error) {
	return t(ctx, u)
}

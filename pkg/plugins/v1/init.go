// Copyright 2020 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"

	"github.com/joelanford/helm-operator/pkg/plugins/internal/kubebuilder/cmdutil"
	"github.com/joelanford/helm-operator/pkg/plugins/v1/chartutil"
	"github.com/joelanford/helm-operator/pkg/plugins/v1/scaffolds"
)

type initSubcommand struct {
	config    config.Config
	apiPlugin createAPISubcommand

	domain      string
	projectName string

	// If true, run the `create api` plugin.
	doCreateAPI bool

	// For help text.
	commandName string
}

var (
	_ plugin.InitSubcommand = &initSubcommand{}
	_ cmdutil.RunOptions    = &initSubcommand{}
)

// UpdateContext define plugin context
func (p *initSubcommand) UpdateContext(ctx *plugin.Context) {
	ctx.Description = `Initialize a new Helm-based operator project.

Writes the following files:
- a helm-charts directory with the chart(s) to build releases from
- a watches.yaml file that defines the mapping between your API and a Helm chart
- a PROJECT file with the domain and project layout configuration
- a Makefile to build the project
- a Kustomization.yaml for customizating manifests
- a Patch file for customizing image for manager manifests
- a Patch file for enabling prometheus metrics
`
	ctx.Examples = fmt.Sprintf(`  $ %[1]s init --plugins=%[2]s \
      --domain=example.com \
      --group=apps \
      --version=v1alpha1 \
      --kind=AppService

  $ %[1]s init --plugins=%[2]s \
      --project-name=myapp
      --domain=example.com \
      --group=apps \
      --version=v1alpha1 \
      --kind=AppService

  $ %[1]s init --plugins=%[2]s \
      --domain=example.com \
      --group=apps \
      --version=v1alpha1 \
      --kind=AppService \
      --helm-chart=myrepo/app

  $ %[1]s init --plugins=%[2]s \
      --domain=example.com \
      --helm-chart=myrepo/app

  $ %[1]s init --plugins=%[2]s \
      --domain=example.com \
      --helm-chart=myrepo/app \
      --helm-chart-version=1.2.3

  $ %[1]s init --plugins=%[2]s \
      --domain=example.com \
      --helm-chart=app \
      --helm-chart-repo=https://charts.mycompany.com/

  $ %[1]s init --plugins=%[2]s \
      --domain=example.com \
      --helm-chart=app \
      --helm-chart-repo=https://charts.mycompany.com/ \
      --helm-chart-version=1.2.3

  $ %[1]s init --plugins=%[2]s \
      --domain=example.com \
      --helm-chart=/path/to/local/chart-directories/app/

  $ %[1]s init --plugins=%[2]s \
      --domain=example.com \
      --helm-chart=/path/to/local/chart-archives/app-1.2.3.tgz
`,
		ctx.CommandName, pluginKey,
	)

	p.commandName = ctx.CommandName
}

// BindFlags will set the flags for the plugin
func (p *initSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.SortFlags = false
	fs.StringVar(&p.domain, "domain", "my.domain", "domain for groups")
	fs.StringVar(&p.projectName, "project-name", "", "name of this project, the default being directory name")
	p.apiPlugin.BindFlags(fs)
}

// InjectConfig will inject the PROJECT file/config in the plugin
func (p *initSubcommand) InjectConfig(c config.Config) {
	// v3 project configs get a 'layout' value.
	_ = c.SetLayout(pluginKey)
	p.config = c
	p.apiPlugin.config = p.config
}

// Run will call the plugin actions
func (p *initSubcommand) Run() error {
	return cmdutil.Run(p)
}

// Validate perform the required validations for this plugin
func (p *initSubcommand) Validate() error {
	// Set values in the config
	if err := p.config.SetProjectName(p.projectName); err != nil {
		return err
	}
	if err := p.config.SetDomain(p.domain); err != nil {
		return err
	}

	// Check if the project name is a valid k8s namespace (DNS 1123 label).
	if p.config.GetProjectName() == "" {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting current directory: %v", err)
		}
		if err = p.config.SetProjectName(strings.ToLower(filepath.Base(dir))); err != nil {
			return err
		}
	}

	if err := validation.IsDNS1123Label(p.config.GetProjectName()); err != nil {
		return fmt.Errorf("project name (%s) is invalid: %v", p.config.GetProjectName(), err)
	}

	defaultOpts := chartutil.CreateOptions{CRDVersion: "v1"}
	if !p.apiPlugin.createOptions.GVK.Empty() || p.apiPlugin.createOptions != defaultOpts {
		p.doCreateAPI = true
		return p.apiPlugin.Validate()
	}

	return nil
}

// GetScaffolder returns cmdutil.Scaffolder which will be executed due the RunOptions interface implementation
func (p *initSubcommand) GetScaffolder() (cmdutil.Scaffolder, error) {
	var (
		apiScaffolder cmdutil.Scaffolder
		err           error
	)
	if p.doCreateAPI {
		apiScaffolder, err = p.apiPlugin.GetScaffolder()
		if err != nil {
			return nil, err
		}
	}
	return scaffolds.NewInitScaffolder(p.config, apiScaffolder), nil
}

// PostScaffold will run the required actions after the default plugin scaffold
func (p *initSubcommand) PostScaffold() error {

	if p.doCreateAPI {
		return p.apiPlugin.PostScaffold()
	}

	fmt.Printf("Next: define a resource with:\n$ %s create api\n", p.commandName)
	return nil
}

// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/statisticsnorway/drone/cmd/drone-server/config"
	"github.com/statisticsnorway/drone/core"
	"github.com/statisticsnorway/drone/livelog"
	"github.com/statisticsnorway/drone/metric/sink"
	"github.com/statisticsnorway/drone/pubsub"
	"github.com/statisticsnorway/drone/service/commit"
	"github.com/statisticsnorway/drone/service/content"
	"github.com/statisticsnorway/drone/service/content/cache"
	"github.com/statisticsnorway/drone/service/hook"
	"github.com/statisticsnorway/drone/service/hook/parser"
	"github.com/statisticsnorway/drone/service/netrc"
	"github.com/statisticsnorway/drone/service/org"
	"github.com/statisticsnorway/drone/service/repo"
	"github.com/statisticsnorway/drone/service/status"
	"github.com/statisticsnorway/drone/service/syncer"
	"github.com/statisticsnorway/drone/service/token"
	"github.com/statisticsnorway/drone/service/user"
	"github.com/statisticsnorway/drone/session"
	"github.com/statisticsnorway/drone/trigger"
	"github.com/statisticsnorway/drone/trigger/cron"
	"github.com/statisticsnorway/drone/version"
	"github.com/drone/go-scm/scm"

	"github.com/google/wire"
)

// wire set for loading the services.
var serviceSet = wire.NewSet(
	commit.New,
	cron.New,
	livelog.New,
	orgs.New,
	parser.New,
	pubsub.New,
	repo.New,
	token.Renewer,
	trigger.New,
	user.New,

	provideContentService,
	provideDatadog,
	provideHookService,
	provideNetrcService,
	provideSession,
	provideStatusService,
	provideSyncer,
	provideSystem,
)

// provideContentService is a Wire provider function that
// returns a contents service wrapped with a simple LRU cache.
func provideContentService(client *scm.Client, renewer core.Renewer) core.FileService {
	return cache.Contents(
		contents.New(client, renewer),
	)
}

// provideHookService is a Wire provider function that returns a
// hook service based on the environment configuration.
func provideHookService(client *scm.Client, renewer core.Renewer, config config.Config) core.HookService {
	return hook.New(client, config.Proxy.Addr, renewer)
}

// provideNetrcService is a Wire provider function that returns
// a netrc service based on the environment configuration.
func provideNetrcService(client *scm.Client, renewer core.Renewer, config config.Config) core.NetrcService {
	return netrc.New(
		client,
		renewer,
		config.Cloning.AlwaysAuth,
		config.Cloning.Username,
		config.Cloning.Password,
	)
}

// provideSession is a Wire provider function that returns a
// user session based on the environment configuration.
func provideSession(store core.UserStore, config config.Config) (core.Session, error) {
	if config.Session.MappingFile != "" {
		return session.Legacy(store, session.Config{
			Secure:      config.Session.Secure,
			Secret:      config.Session.Secret,
			Timeout:     config.Session.Timeout,
			MappingFile: config.Session.MappingFile,
		})
	}

	return session.New(store, session.NewConfig(
		config.Session.Secret,
		config.Session.Timeout,
		config.Session.Secure),
	), nil
}

// provideUserService is a Wire provider function that returns a
// user service based on the environment configuration.
func provideStatusService(client *scm.Client, renewer core.Renewer, config config.Config) core.StatusService {
	return status.New(client, renewer, status.Config{
		Base:     config.Server.Addr,
		Name:     config.Status.Name,
		Disabled: config.Status.Disabled,
	})
}

// provideSyncer is a Wire provider function that returns a
// repository synchronizer.
func provideSyncer(repoz core.RepositoryService,
	repos core.RepositoryStore,
	users core.UserStore,
	batch core.Batcher,
	config config.Config) core.Syncer {
	sync := syncer.New(repoz, repos, users, batch)
	// the user can define a filter that limits which
	// repositories can be synchronized and stored in the
	// database.
	if filter := config.Repository.Filter; len(filter) > 0 {
		sync.SetFilter(syncer.NamespaceFilter(filter))
	}
	return sync
}

// provideSyncer is a Wire provider function that returns the
// system details structure.
func provideSystem(config config.Config) *core.System {
	return &core.System{
		Proto:   config.Server.Proto,
		Host:    config.Server.Host,
		Link:    config.Server.Addr,
		Version: version.Version.String(),
	}
}

// provideDatadog is a Wire provider function that returns the
// datadog sink.
func provideDatadog(
	users core.UserStore,
	repos core.RepositoryStore,
	builds core.BuildStore,
	system *core.System,
	license *core.License,
	config config.Config,
) *sink.Datadog {
	return sink.New(
		users,
		repos,
		builds,
		*system,
		sink.Config{
			Endpoint:         config.Datadog.Endpoint,
			Token:            config.Datadog.Token,
			License:          license.Kind,
			Licensor:         license.Licensor,
			Subscription:     license.Subscription,
			EnableGithub:     config.IsGitHub(),
			EnableGithubEnt:  config.IsGitHubEnterprise(),
			EnableGitlab:     config.IsGitLab(),
			EnableBitbucket:  config.IsBitbucket(),
			EnableStash:      config.IsStash(),
			EnableGogs:       config.IsGogs(),
			EnableGitea:      config.IsGitea(),
			EnableAgents:     config.Agent.Enabled,
			EnableNomad:      config.Nomad.Enabled,
			EnableKubernetes: config.Kube.Enabled,
		},
	)
}

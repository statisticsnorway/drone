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

package user

import (
	"context"

	"github.com/statisticsnorway/drone/core"
	"github.com/drone/go-scm/scm"
)

type service struct {
	client *scm.Client
}

// New returns a new User service that provides access to
// user data from the source code management system.
func New(client *scm.Client) core.UserService {
	return &service{client: client}
}

func (s *service) Find(ctx context.Context, access, refresh string) (*core.User, error) {
	ctx = context.WithValue(ctx, scm.TokenKey{}, &scm.Token{
		Token:   access,
		Refresh: refresh,
	})
	src, _, err := s.client.Users.Find(ctx)
	if err != nil {
		return nil, err
	}
	dst := &core.User{
		Login:  src.Login,
		Email:  src.Email,
		Avatar: src.Avatar,
	}
	if !src.Created.IsZero() {
		dst.Created = src.Created.Unix()
	}
	if !src.Updated.IsZero() {
		dst.Updated = src.Updated.Unix()
	}
	return dst, nil
}

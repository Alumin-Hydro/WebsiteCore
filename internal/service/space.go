// Copyright 2022 ROC. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package service

import (
	"fmt"

	"github.com/BZYA-Community/WebsiteCore/internal/conf"
	"github.com/BZYA-Community/WebsiteCore/internal/servants"
	"github.com/Masterminds/semver/v3"
	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
)

type spaceXService struct {
	*baseHttpService
}

func (s *spaceXService) Name() string {
	return "SpaceXService"
}

func (s *spaceXService) Version() *semver.Version {
	return semver.MustParse("v0.1.0")
}

func (s *spaceXService) OnInit() error {
	s.registerRoute(s, servants.RegisterSpaceXServants)
	return nil
}

func (s *spaceXService) String() string {
	return fmt.Sprintf("listen on %s\n", color.GreenString("http://%s:%s", conf.SpaceXServerSetting.HttpIp, conf.SpaceXServerSetting.HttpPort))
}

func newSpaceXEngine() *gin.Engine {
	return newHTTPEngine(httpEngineOptions{API: true})
}

func newSpaceXService() Service {
	server := sharedHTTPServer(conf.SpaceXServerSetting, newSpaceXEngine)
	return &spaceXService{
		baseHttpService: &baseHttpService{
			server: server,
		},
	}
}

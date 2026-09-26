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

type localossService struct {
	*baseHttpService
}

func (s *localossService) Name() string {
	return "LocalossService"
}

func (s *localossService) Version() *semver.Version {
	return semver.MustParse("v0.1.0")
}

func (s *localossService) OnInit() error {
	s.registerRoute(s, servants.RegisterLocalossServants)
	return nil
}

func (s *localossService) String() string {
	return fmt.Sprintf("listen on %s\n", color.GreenString("http://%s:%s", conf.LocalossServerSetting.HttpIp, conf.LocalossServerSetting.HttpPort))
}

func newLocalossEngine() *gin.Engine {
	return newHTTPEngine(httpEngineOptions{})
}

func newLocalossService() Service {
	server := sharedHTTPServer(conf.LocalossServerSetting, newLocalossEngine)
	return &localossService{
		baseHttpService: &baseHttpService{
			server: server,
		},
	}
}

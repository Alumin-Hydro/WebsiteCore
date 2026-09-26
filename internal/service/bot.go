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

type botService struct {
	*baseHttpService
}

func (s *botService) Name() string {
	return "BotService"
}

func (s *botService) Version() *semver.Version {
	return semver.MustParse("v0.1.0")
}

func (s *botService) OnInit() error {
	s.registerRoute(s, servants.RegisterBotServants)
	return nil
}

func (s *botService) String() string {
	return fmt.Sprintf("listen on %s\n", color.GreenString("http://%s:%s", conf.BotServerSetting.HttpIp, conf.BotServerSetting.HttpPort))
}

func newBotEngine() *gin.Engine {
	return newHTTPEngine(httpEngineOptions{API: true})
}

func newBotService() Service {
	server := sharedHTTPServer(conf.BotServerSetting, newBotEngine)
	return &botService{
		baseHttpService: &baseHttpService{
			server: server,
		},
	}
}

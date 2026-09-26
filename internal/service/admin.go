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

type adminService struct {
	*baseHttpService
}

func (s *adminService) Name() string {
	return "AdminService"
}

func (s *adminService) Version() *semver.Version {
	return semver.MustParse("v0.1.0")
}

func (s *adminService) OnInit() error {
	s.registerRoute(s, servants.RegisterAdminServants)
	return nil
}

func (s *adminService) String() string {
	return fmt.Sprintf("listen on %s\n", color.GreenString("http://%s:%s", conf.AdminServerSetting.HttpIp, conf.AdminServerSetting.HttpPort))
}

func newAdminEngine() *gin.Engine {
	return newHTTPEngine(httpEngineOptions{API: true})
}

func newAdminService() Service {
	server := sharedHTTPServer(conf.AdminServerSetting, newAdminEngine)
	return &adminService{
		baseHttpService: &baseHttpService{
			server: server,
		},
	}
}

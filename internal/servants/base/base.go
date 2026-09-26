// Copyright 2022 ROC. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package base

import (
	"context"
	"fmt"
	"math"
	"net/http"

	"github.com/BZYA-Community/WebsiteCore/internal/application/content"
	"github.com/BZYA-Community/WebsiteCore/internal/conf"
	"github.com/BZYA-Community/WebsiteCore/internal/core"
	"github.com/BZYA-Community/WebsiteCore/internal/core/ms"
	"github.com/BZYA-Community/WebsiteCore/internal/dao"
	"github.com/BZYA-Community/WebsiteCore/internal/dao/cache"
	"github.com/BZYA-Community/WebsiteCore/internal/infra/events"
	"github.com/BZYA-Community/WebsiteCore/internal/model/joint"
	"github.com/BZYA-Community/WebsiteCore/pkg/app"
	"github.com/BZYA-Community/WebsiteCore/pkg/xerror"
	"github.com/alimy/mir/v5"
	"github.com/cockroachdb/errors"
	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type BaseServant struct {
	bindAny  func(c *gin.Context, obj any) error
	bindJson func(c *gin.Context, obj any) error
}

type DaoServant struct {
	*BaseServant
	*content.Views

	Dsa   core.WebDataServantA
	Ds    core.DataService
	Ts    core.TweetSearchService
	Redis core.RedisCache
}

type SentryHubSetter interface {
	SetSentryHub(hub *sentry.Hub)
}

type UserSetter interface {
	SetUser(*ms.User)
}

type UserIdSetter interface {
	SetUserId(int64)
}

type PageInfoSetter interface {
	SetPageInfo(page, pageSize int)
}

func UserFrom(c *gin.Context) (*ms.User, bool) {
	if u, exists := c.Get("USER"); exists {
		user, ok := u.(*ms.User)
		return user, ok
	}
	return nil, false
}

func UserIdFrom(c *gin.Context) (int64, bool) {
	if uid, exists := c.Get("UID"); exists {
		v, ok := uid.(int64)
		return v, ok
	}
	return -1, false
}

func UserNameFrom(c *gin.Context) (string, bool) {
	if username, exists := c.Get("USERNAME"); exists {
		v, ok := username.(string)
		return v, ok
	}
	return "", false
}

func bindAny(c *gin.Context, obj any) error {
	var errs xerror.ValidErrors
	err := c.ShouldBind(obj)
	if err != nil {
		// 逐字段收集入参校验错误明细，便于客户端定位具体出错字段
		if validationErrs, ok := err.(validator.ValidationErrors); ok {
			for _, fieldErr := range validationErrs {
				errs = append(errs, &xerror.ValidError{
					Message: fmt.Sprintf("字段 %s 校验失败: %s", fieldErr.Field(), fieldErr.Tag()),
				})
			}
		} else {
			errs = append(errs, &xerror.ValidError{Message: err.Error()})
		}
		return mir.NewError(xerror.InvalidParams.StatusCode(), xerror.InvalidParams.WithDetails(errs.Errors()...))
	}
	// setup *core.User if needed
	if setter, ok := obj.(UserSetter); ok {
		user, _ := UserFrom(c)
		setter.SetUser(user)
	}
	// setup UserId if needed
	if setter, ok := obj.(UserIdSetter); ok {
		uid, _ := UserIdFrom(c)
		setter.SetUserId(uid)
	}
	// setup PageInfo if needed
	if setter, ok := obj.(PageInfoSetter); ok {
		page, pageSize := app.GetPageInfo(c)
		setter.SetPageInfo(page, pageSize)
	}
	return nil
}

func bindAnySentry(c *gin.Context, obj any) error {
	hub := sentrygin.GetHubFromContext(c)
	var errs xerror.ValidErrors
	err := c.ShouldBind(obj)
	if err != nil {
		xerr := mir.NewError(xerror.InvalidParams.StatusCode(), xerror.InvalidParams.WithDetails(errs.Error()))
		if hub != nil {
			hub.CaptureException(errors.Wrap(xerr, "bind object"))
		}
		return xerr
	}
	// setup sentry hub if needed
	if setter, ok := obj.(SentryHubSetter); ok && hub != nil {
		setter.SetSentryHub(hub)
	}
	// setup *core.User if needed
	if setter, ok := obj.(UserSetter); ok {
		user, _ := UserFrom(c)
		setter.SetUser(user)
	}
	// setup UserId if needed
	if setter, ok := obj.(UserIdSetter); ok {
		uid, _ := UserIdFrom(c)
		setter.SetUserId(uid)
	}
	// setup PageInfo if needed
	if setter, ok := obj.(PageInfoSetter); ok {
		page, pageSize := app.GetPageInfo(c)
		setter.SetPageInfo(page, pageSize)
	}
	return nil
}

func RenderAny(c *gin.Context, data any, err error) {
	if err == nil {
		c.JSON(http.StatusOK, &joint.JsonResp{
			Code: 0,
			Msg:  "success",
			Data: data,
		})
	} else {
		statusCode, code := xerror.HttpStatusCode(err)
		c.JSON(statusCode, &joint.JsonResp{
			Code: code,
			Msg:  err.Error(),
		})
	}
}

func (s *BaseServant) Bind(c *gin.Context, obj any) error {
	return s.bindAny(c, obj)
}

func (s *BaseServant) BindJson(c *gin.Context, obj any) error {
	return s.bindJson(c, obj)
}

func (s *BaseServant) Render(c *gin.Context, data any, err error) {
	if err == nil {
		c.JSON(http.StatusOK, &joint.JsonResp{
			Code: 0,
			Msg:  "success",
			Data: data,
		})
	} else {
		statusCode, code := xerror.HttpStatusCode(err)
		c.JSON(statusCode, &joint.JsonResp{
			Code: code,
			Msg:  err.Error(),
		})
	}
}

func (s *DaoServant) PushAllPostToSearch() {
	events.OnEvent(&pushAllPostToSearchEvent{
		fn: s.pushAllPostToSearch,
	})
}

func (s *DaoServant) pushAllPostToSearch() error {
	ctx := context.Background()
	if err := s.Redis.SetPushToSearchJob(ctx); err == nil {
		defer s.Redis.DelPushToSearchJob(ctx)
		splitNum := 1000
		posts, totalRows, err := s.Ds.ListSyncSearchTweets(splitNum, 0)
		if err != nil {
			return fmt.Errorf("get first page tweets push to search failed: %s", err)
		}
		i, nums := 0, int(math.Ceil(float64(totalRows)/float64(splitNum)))
		for {
			postsFormated, xerr := s.Ds.MergePosts(posts)
			if xerr != nil || len(posts) != len(postsFormated) {
				continue
			}
			for i, pf := range postsFormated {
				contentFormated := ""
				for _, content := range pf.Contents {
					if content.Type == ms.ContentTypeText || content.Type == ms.ContentTypeTitle || content.Type == ms.ContentTypeMarkdown {
						contentFormated = contentFormated + content.Content + "\n"
					}
				}
				docs := []core.TsDocItem{{
					Post:    posts[i],
					Content: contentFormated,
				}}
				s.Ts.AddDocuments(docs, fmt.Sprintf("%d", posts[i].ID))
			}
			if i++; i >= nums {
				break
			}
			if posts, _, err = s.Ds.ListSyncSearchTweets(splitNum, i*splitNum); err != nil {
				return fmt.Errorf("get tweets push to search failed: %s, limit[%d] offset[%d]", err, splitNum, i*splitNum)
			}
		}
	} else {
		return fmt.Errorf("redis: set JOB_PUSH_TO_SEARCH error: %w", err)
	}
	return nil
}

func (s *DaoServant) PushPostToSearch(post *ms.Post) {
	events.OnEvent(&pushPostToSearchEvent{
		fn:   s.pushPostToSearch,
		post: post,
	})
}

func (s *DaoServant) pushPostToSearch(post *ms.Post) {
	postFormated := post.Format()
	postFormated.User = &ms.UserFormated{
		ID: post.UserID,
	}
	contents, _ := s.Ds.GetPostContentsByIDs([]int64{post.ID})
	for _, content := range contents {
		postFormated.Contents = append(postFormated.Contents, content.Format())
	}
	contentFormated := ""
	for _, content := range postFormated.Contents {
		if content.Type == ms.ContentTypeText || content.Type == ms.ContentTypeTitle || content.Type == ms.ContentTypeMarkdown {
			contentFormated = contentFormated + content.Content + "\n"
		}
	}
	docs := []core.TsDocItem{{
		Post:    post,
		Content: contentFormated,
	}}
	s.Ts.AddDocuments(docs, fmt.Sprintf("%d", post.ID))
}

func (s *DaoServant) DeleteSearchPost(post *ms.Post) error {
	return s.Ts.DeleteDocuments([]string{fmt.Sprintf("%d", post.ID)})
}

func NewBindAnyFn() func(c *gin.Context, obj any) error {
	if conf.UseSentryGin() {
		return bindAnySentry
	}
	return bindAny
}

func NewBindJsonFn() func(c *gin.Context, obj any) error {
	if conf.UseSentryGin() {
		return bindAnySentry
	}
	return bindAny
}

func NewBaseServant() *BaseServant {
	return &BaseServant{
		bindAny:  NewBindAnyFn(),
		bindJson: NewBindJsonFn(),
	}
}

func NewDaoServant() *DaoServant {
	ds := dao.DataService()
	return &DaoServant{
		BaseServant: NewBaseServant(),
		Redis:       cache.NewRedisCache(),
		Dsa:         dao.WebDataServantA(),
		Ds:          ds,
		Views:       content.New(ds),
		Ts:          dao.TweetSearchService(),
	}
}

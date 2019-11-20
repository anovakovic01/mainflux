package api

import (
	"context"
	"fmt"
	"time"

	"github.com/mainflux/mainflux/authz"
	log "github.com/mainflux/mainflux/logger"
)

var _ authz.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    authz.Service
}

// LoggingMiddleware returns given service instance wrappend in the logging
// middleware.
func LoggingMiddleware(svc authz.Service, logger log.Logger) authz.Service {
	return loggingMiddleware{
		logger: logger,
		svc:    svc,
	}
}

func (lm loggingMiddleware) Authorize(ctx context.Context, p authz.Policy) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method authorize for subject %s, object %s and action %s took %s to complete", p.Subject, p.Object, p.Action, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Authorize(ctx, p)
}

func (lm loggingMiddleware) Connect(ctx context.Context, token string, ps map[string]authz.Policy) (errs map[string]error, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method connect took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Connect(ctx, token, ps)
}

func (lm loggingMiddleware) Disconnect(ctx context.Context, token string, ps map[string]authz.Policy) (errs map[string]error, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method disconnect took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Disconnect(ctx, token, ps)
}

func (lm loggingMiddleware) AddThings(ctx context.Context, owner string, ids ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method add_things for owner %s took %s to complete", owner, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AddThings(ctx, owner, ids...)
}

func (lm loggingMiddleware) AddChannels(ctx context.Context, owner string, ids ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method add_channels for owner %s took %s to complete", owner, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AddChannels(ctx, owner, ids...)
}

func (lm loggingMiddleware) RemoveThing(ctx context.Context, owner string, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_thing for owner %s and thing %s took %s to complete", owner, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveThing(ctx, owner, id)
}

func (lm loggingMiddleware) RemoveChannel(ctx context.Context, owner string, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_channel for owner %s and thing %s took %s to complete", owner, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveThing(ctx, owner, id)
}

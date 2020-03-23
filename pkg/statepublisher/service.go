package servicestatepublisher

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/einride/clock-go/pkg/clock"
	diagnosticsv1beta1 "github.com/einride/proto/gen/go/einride/drive/diagnostics/v1beta1"
	"github.com/einride/supervisor-go"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	Enabled                bool
	CommitSHAEnvKey        string
	GithubRepositoryEnvKey string
	LoopInterval           time.Duration
}

// ConfigJSON is a byte slice that contains the JSON marshaled
type ConfigJSON []byte

type ProtoPublisher interface {
	TransmitProto(ctx context.Context, message proto.Message) error
	Close() error
}

type ProtoPublisherProvider func(context.Context) (ProtoPublisher, error)

type Service struct {
	logger                 *zap.Logger
	Clock                  clock.Clock
	ConfigJSON             []byte
	Commit                 string
	Repo                   string
	LoopInterval           time.Duration
	ProtoPublisherProvider ProtoPublisherProvider
	mutex                  sync.Mutex
	lastStatusUpdate       *timestamp.Timestamp
}

func Init(
	logger *zap.Logger,
	clock clock.Clock,
	cfg *Config,
	cfgJSON ConfigJSON,
	provider ProtoPublisherProvider,
) (*Service, error) {
	if !cfg.Enabled {
		logger.Info("state publisher not enabled", zap.Any("cfg", cfg))
		return nil, nil
	}
	commitSHA, ok := os.LookupEnv(cfg.CommitSHAEnvKey)
	if !ok {
		return nil, errors.New("could not find commit sha env")
	}
	repo, ok := os.LookupEnv(cfg.GithubRepositoryEnvKey)
	if !ok {
		return nil, errors.New("could not find github repo env")
	}
	return &Service{
		ProtoPublisherProvider: provider,
		logger:                 logger,
		Clock:                  clock,
		Commit:                 commitSHA,
		ConfigJSON:             cfgJSON,
		Repo:                   repo,
		LoopInterval:           cfg.LoopInterval,
	}, nil
}

func (s *Service) StatusInsertChannel(updates []supervisor.StatusUpdate) {
	var newest supervisor.StatusUpdate
	for _, u := range updates {
		if newest.Time.After(u.Time) {
			continue
		}
		newest = u
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var err error
	if s.lastStatusUpdate, err = ptypes.TimestampProto(newest.Time); err != nil {
		s.logger.Error("status insert channel", zap.Error(err))
	}
}

func (s *Service) Run(ctx context.Context) error {
	publisher, err := s.ProtoPublisherProvider(ctx)
	if err != nil {
		return errors.Wrap(err, "run state publisher")
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		<-ctx.Done()
		return publisher.Close()
	})
	g.Go(func() error {
		s.logger.Debug("running")
		return s.run(ctx, publisher, s.Clock.NewTicker(s.LoopInterval))
	})
	defer s.logger.Debug("stopped")
	if err := g.Wait(); err != nil && !strings.Contains(err.Error(), "closed") {
		return errors.Wrapf(err, "AD state publisher")
	}
	return nil
}

func (s *Service) run(ctx context.Context, publisher ProtoPublisher, ticker clock.Ticker) error {
	defer ticker.Stop()
	ticks := ticker.C()
	ctxDone := ctx.Done()
	s.logger.Info("running service state publisher")
	for {
		s.mutex.Lock()
		lastRestartTime := s.lastStatusUpdate
		s.mutex.Unlock()
		msg := &diagnosticsv1beta1.ServiceStatus{
			Repo:            s.Repo,
			Commit:          s.Commit,
			ConfigJson:      s.ConfigJSON,
			PublishTime:     s.Clock.NowProto(),
			LastRestartTime: lastRestartTime,
		}
		select {
		case <-ctxDone:
			return nil
		case <-ticks:
			s.logger.Debug("transmitting service status", zap.Any("msg", msg))
			err := publisher.TransmitProto(context.Background(), msg)
			if err != nil {
				return errors.Wrap(err, "init state publisher")
			}
		}
	}
}

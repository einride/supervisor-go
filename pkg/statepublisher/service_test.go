package servicestatepublisher

import (
	"context"
	"os"
	"testing"

	"github.com/einride/clock-go/pkg/clock"
	"github.com/einride/supervisor-go"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type testTransmitter struct {
	callback func(proto.Message)
}

func (t *testTransmitter) TransmitProto(ctx context.Context, message proto.Message) error {
	t.callback(message)
	return nil
}

func (t *testTransmitter) Close() error {
	return nil
}

func TestStartSequence(t *testing.T) {
	cfg := &Config{
		Enabled:                true,
		CommitSHAEnvKey:        "123",
		GithubRepositoryEnvKey: "apa",
		LoopInterval:           1,
	}
	require.NoError(t, os.Setenv(cfg.CommitSHAEnvKey, "foo"))
	require.NoError(t, os.Setenv(cfg.GithubRepositoryEnvKey, "bar"))
	ctx, cancel := context.WithCancel(context.Background())
	var e errgroup.Group
	var p ProtoPublisher = &testTransmitter{
		callback: func(message proto.Message) {
			cancel()
		},
	}
	c := clock.System()
	s, err := Init(zap.NewExample(), c, cfg, []byte("hej"),
		func(context.Context) (ProtoPublisher, error) {
			return p, nil
		},
	)
	require.NoError(t, err)
	e.Go(func() error {
		return s.Run(ctx)
	})
	s.StatusInsertChannel([]supervisor.StatusUpdate{
		{
			ServiceID:   1,
			ServiceName: "bepa",
			Time:        c.Now(),
			Status:      123,
			Err:         errors.New("err"),
		},
	})
	require.NoError(t, e.Wait())
}

func TestStatusUpdateTooEarly(t *testing.T) {
	cfg := &Config{
		Enabled:                true,
		CommitSHAEnvKey:        "123",
		GithubRepositoryEnvKey: "apa",
		LoopInterval:           1,
	}
	require.NoError(t, os.Setenv(cfg.CommitSHAEnvKey, "foo"))
	require.NoError(t, os.Setenv(cfg.GithubRepositoryEnvKey, "bar"))
	_, cancel := context.WithCancel(context.Background())
	var e errgroup.Group
	var p ProtoPublisher = &testTransmitter{
		callback: func(message proto.Message) {
			cancel()
		},
	}
	c := clock.System()
	s, err := Init(zap.NewExample(), c, cfg, []byte("hej"),
		func(context.Context) (ProtoPublisher, error) {
			return p, nil
		},
	)
	require.NoError(t, err)
	s.StatusInsertChannel([]supervisor.StatusUpdate{
		{
			ServiceID:   1,
			ServiceName: "bepa",
			Time:        c.Now(),
			Status:      123,
			Err:         errors.New("err"),
		},
	})
	require.NoError(t, e.Wait())
}

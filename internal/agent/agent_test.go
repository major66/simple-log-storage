package agent_test

import (
	"context"
	"crypto/tls"
	"fmt"
	api "github.com/major66/simple-log-storage/api/v1"
	"github.com/major66/simple-log-storage/internal/agent"
	"github.com/major66/simple-log-storage/internal/config"
	dynaport "github.com/major66/simple-log-storage/internal/dynaPort"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestAgent(t *testing.T) {
	serverTLSConfig, err := config.SetupTLSConfig(
		config.TLSConfig{
			CertFile:      config.ServerCertFile,
			KeyFile:       config.ServerKeyFile,
			CAFile:        config.CAFile,
			Server:        true,
			ServerAddress: "127.0.0.1",
		})
	require.NoError(t, err)
	peerTLSConfig, err := config.SetupTLSConfig(
		config.TLSConfig{
			CertFile:      config.RootClientCertFile,
			KeyFile:       config.RootClientKeyFile,
			CAFile:        config.CAFile,
			Server:        false,
			ServerAddress: "127.0.0.1",
		})
	require.NoError(t, err)
	var agents []*agent.Agent
	for i := 0; i < 3; i++ {
		ports := dynaport.Get(2)
		bindAddr := fmt.Sprintf("%s:%d", "127.0.0.1", ports[0])
		rpcPort := ports[1]
		dataDir, err := ioutil.TempDir("", "agent-test-log")
		require.NoError(t, err)
		var startJoinAddrs []string
		if i != 0 {
			startJoinAddrs = append(startJoinAddrs, agents[0].Config.BindAddr)
		}
		agent, err := agent.New(
			agent.Config{
				NodeName:        fmt.Sprintf("%d", i),
				StartJoinAddrs:  startJoinAddrs,
				BindAddr:        bindAddr,
				RPCPort:         rpcPort,
				DataDir:         dataDir,
				ACLModelFile:    config.ACLModelFile,
				ACLPolicyFile:   config.ACLPolicyFile,
				ServerTLSConfig: serverTLSConfig,
				PeerTLSConfig:   peerTLSConfig,
				Bootstrap:       i == 0,
			})
		require.NoError(t, err)
		agents = append(agents, agent)
	}
	defer func() {
		for _, agent := range agents {
			err := agent.Shutdown()
			require.NoError(t, err)
			require.NoError(t, os.RemoveAll(agent.Config.DataDir))
		}
	}()
	time.Sleep(3 * time.Second)

	leaderClient := client(t, agents[0], peerTLSConfig)
	produceResponse, err := leaderClient.Produce(context.Background(), &api.ProduceRequest{Record: &api.Record{Value: []byte("foo")}})
	require.NoError(t, err)
	consumeResponse, err := leaderClient.Consume(context.Background(), &api.ConsumeRequest{Offset: produceResponse.Offset})
	require.NoError(t, err)
	require.Equal(t, consumeResponse.Record.Value, []byte("foo"))

	// wait until replication has finished
	time.Sleep(3 * time.Second)
	followerClient := client(t, agents[1], peerTLSConfig)
	consumeResponse, err = followerClient.Consume(context.Background(), &api.ConsumeRequest{Offset: produceResponse.Offset})
	require.NoError(t, err)
	require.Equal(t, consumeResponse.Record.Value, []byte("foo"))

	consumeResponse, err = leaderClient.Consume(context.Background(), &api.ConsumeRequest{Offset: produceResponse.Offset + 1})
	require.Nil(t, consumeResponse)
	require.Error(t, err)
	got := grpc.Code(err)
	want := grpc.Code(api.ErrorOffsetOutOfRange{}.GRPCStatus().Err())
	require.Equal(t, got, want)
}

func client(t *testing.T, agent *agent.Agent, tlsConfig *tls.Config) api.LogClient {
	tlsCreds := credentials.NewTLS(tlsConfig)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}
	rpcAddr, err := agent.Config.RPCAddr()
	require.NoError(t, err)
	conn, err := grpc.Dial(fmt.Sprintf("%s", rpcAddr), opts...)
	require.NoError(t, err)
	client := api.NewLogClient(conn)
	return client
}
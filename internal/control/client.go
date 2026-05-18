// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

package control

import (
	"context"

	buoyv1 "github.com/PharosVPN/helm/internal/gen/pharos/buoy/v1"
	"google.golang.org/grpc"
)

// Client is helm's control connection to one buoy node. It is safe for
// concurrent use; close it when done.
type Client struct {
	cc  *grpc.ClientConn
	rpc buoyv1.NodeControlClient
}

// Close releases the connection.
func (c *Client) Close() error { return c.cc.Close() }

// Status reports node and per-protocol service health.
func (c *Client) Status(ctx context.Context) (*buoyv1.GetStatusResponse, error) {
	return c.rpc.GetStatus(ctx, &buoyv1.GetStatusRequest{})
}

// Metrics reports the node's counters for a metrics sample.
func (c *Client) Metrics(ctx context.Context) (*buoyv1.GetMetricsResponse, error) {
	return c.rpc.GetMetrics(ctx, &buoyv1.GetMetricsRequest{})
}

// PushConfig replaces the data-plane config for one protocol.
func (c *Client) PushConfig(ctx context.Context, protocol buoyv1.Protocol, revision int64, config []byte) (*buoyv1.PushConfigResponse, error) {
	return c.rpc.PushConfig(ctx, &buoyv1.PushConfigRequest{
		Protocol: protocol,
		Revision: revision,
		Config:   config,
	})
}

// AddPeer adds a single peer live.
func (c *Client) AddPeer(ctx context.Context, peer *buoyv1.Peer) (*buoyv1.PeerResponse, error) {
	return c.rpc.AddPeer(ctx, &buoyv1.AddPeerRequest{Peer: peer})
}

// RemovePeer revokes a single peer live.
func (c *Client) RemovePeer(ctx context.Context, protocol buoyv1.Protocol, publicKey string) (*buoyv1.PeerResponse, error) {
	return c.rpc.RemovePeer(ctx, &buoyv1.RemovePeerRequest{
		Protocol:  protocol,
		PublicKey: publicKey,
	})
}

// ListPeers returns configured peers and their runtime state. A protocol of
// PROTOCOL_UNSPECIFIED returns every peer.
func (c *Client) ListPeers(ctx context.Context, protocol buoyv1.Protocol) (*buoyv1.ListPeersResponse, error) {
	return c.rpc.ListPeers(ctx, &buoyv1.ListPeersRequest{Protocol: protocol})
}

// RestartService restarts one protocol's data-plane service on the node.
func (c *Client) RestartService(ctx context.Context, protocol buoyv1.Protocol) (*buoyv1.RestartServiceResponse, error) {
	return c.rpc.RestartService(ctx, &buoyv1.RestartServiceRequest{Protocol: protocol})
}

// WatchEvents opens the node's live event server-stream. The caller reads
// events with Recv until ctx is cancelled or the stream ends.
func (c *Client) WatchEvents(ctx context.Context) (grpc.ServerStreamingClient[buoyv1.Event], error) {
	return c.rpc.WatchEvents(ctx, &buoyv1.WatchEventsRequest{})
}

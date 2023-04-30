// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package net

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ipfs/boxo/bitswap"
	"github.com/ipfs/boxo/bitswap/network"
	"github.com/ipfs/boxo/blockservice"
	exchange "github.com/ipfs/boxo/exchange"
	dag "github.com/ipfs/boxo/ipld/merkledag"
	"github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	ipld "github.com/ipfs/go-ipld-format"
	gostream "github.com/libp2p/go-libp2p-gostream"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	peerstore "github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/routing"
	ma "github.com/multiformats/go-multiaddr"
	"google.golang.org/grpc"

	"github.com/sourcenetwork/defradb/client"
	"github.com/sourcenetwork/defradb/core"
	corenet "github.com/sourcenetwork/defradb/core/net"
	"github.com/sourcenetwork/defradb/datastore"
	"github.com/sourcenetwork/defradb/errors"
	"github.com/sourcenetwork/defradb/events"
	"github.com/sourcenetwork/defradb/logging"
	"github.com/sourcenetwork/defradb/merkle/clock"
	pb "github.com/sourcenetwork/defradb/net/pb"
)

var (
	numWorkers = 5
)

// Peer is a DefraDB Peer node which exposes all the LibP2P host/peer functionality
// to the underlying DefraDB instance.
type Peer struct {
	//config??

	db            client.DB
	updateChannel chan events.Update

	host host.Host
	dht  routing.Routing
	ps   *pubsub.PubSub

	server *server
	p2pRPC *grpc.Server // rpc server over the p2p network

	// Used to close the dagWorker pool for a given document.
	// The string represents a dockey.
	closeJob chan string
	sendJobs chan *dagJob

	// outstanding log request currently being processed
	queuedChildren *cidSafeSet

	// replicators is a map from collectionName => peerId
	replicators map[string]map[peer.ID]struct{}
	mu          sync.Mutex

	// peer DAG service
	ipld.DAGService
	exch  exchange.Interface
	bserv blockservice.BlockService

	ctx    context.Context
	cancel context.CancelFunc
}

// NewPeer creates a new instance of the DefraDB server as a peer-to-peer node.
func NewPeer(
	ctx context.Context,
	db client.DB,
	h host.Host,
	dht routing.Routing,
	ps *pubsub.PubSub,
	tcpAddr ma.Multiaddr,
	serverOptions []grpc.ServerOption,
	dialOptions []grpc.DialOption,
) (*Peer, error) {
	if db == nil {
		return nil, errors.New("database object can't be empty")
	}

	ctx, cancel := context.WithCancel(ctx)
	p := &Peer{
		host:           h,
		dht:            dht,
		ps:             ps,
		db:             db,
		p2pRPC:         grpc.NewServer(serverOptions...),
		ctx:            ctx,
		cancel:         cancel,
		closeJob:       make(chan string),
		sendJobs:       make(chan *dagJob),
		replicators:    make(map[string]map[peer.ID]struct{}),
		queuedChildren: newCidSafeSet(),
	}
	var err error
	p.server, err = newServer(p, db, dialOptions...)
	if err != nil {
		return nil, err
	}

	err = p.loadReplicators(p.ctx)
	if err != nil {
		return nil, err
	}

	p.setupBlockService()
	p.setupDAGService()

	return p, nil
}

// Start all the internal workers/goroutines/loops that manage the P2P state.
func (p *Peer) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p2plistener, err := gostream.Listen(p.host, corenet.Protocol)
	if err != nil {
		return err
	}

	if p.ps != nil {
		if !p.db.Events().Updates.HasValue() {
			return errors.New("tried to subscribe to update channel, but update channel is nil")
		}

		updateChannel, err := p.db.Events().Updates.Value().Subscribe()
		if err != nil {
			return err
		}
		p.updateChannel = updateChannel

		log.Info(p.ctx, "Starting internal broadcaster for pubsub network")
		go p.handleBroadcastLoop()
	}

	// register the p2p gRPC server
	go func() {
		pb.RegisterServiceServer(p.p2pRPC, p.server)
		if err := p.p2pRPC.Serve(p2plistener); err != nil &&
			!errors.Is(err, grpc.ErrServerStopped) {
			log.FatalE(p.ctx, "Fatal P2P RPC server error", err)
		}
	}()

	// start sendJobWorker
	go p.sendJobWorker()

	return nil
}

// Close the peer node and all its internal workers/goroutines/loops.
func (p *Peer) Close() error {
	// close topics
	if err := p.server.removeAllPubsubTopics(); err != nil {
		log.ErrorE(p.ctx, "Error closing pubsub topics", err)
	}

	// stop gRPC server
	for _, c := range p.server.conns {
		if err := c.Close(); err != nil {
			log.ErrorE(p.ctx, "Failed closing server RPC connections", err)
		}
	}
	stopGRPCServer(p.ctx, p.p2pRPC)
	// stopGRPCServer(p.tcpRPC)

	// close event emitters
	if p.server.pubSubEmitter != nil {
		if err := p.server.pubSubEmitter.Close(); err != nil {
			log.Info(p.ctx, "Could not close pubsub event emitter", logging.NewKV("Error", err.Error()))
		}
	}
	if p.server.pushLogEmitter != nil {
		if err := p.server.pushLogEmitter.Close(); err != nil {
			log.Info(p.ctx, "Could not close push log event emitter", logging.NewKV("Error", err.Error()))
		}
	}

	if p.db.Events().Updates.HasValue() {
		p.db.Events().Updates.Value().Unsubscribe(p.updateChannel)
	}

	if err := p.bserv.Close(); err != nil {
		log.ErrorE(p.ctx, "Error closing block service", err)
	}

	p.cancel()
	return nil
}

// handleBroadcast loop manages the transition of messages
// from the internal broadcaster to the external pubsub network
func (p *Peer) handleBroadcastLoop() {
	log.Debug(p.ctx, "Waiting for messages on internal broadcaster")
	for {
		log.Debug(p.ctx, "Handling internal broadcast bus message")
		update, isOpen := <-p.updateChannel
		if !isOpen {
			return
		}

		// check log priority, 1 is new doc log
		// 2 is update log
		var err error
		if update.Priority == 1 {
			err = p.handleDocCreateLog(update)
		} else if update.Priority > 1 {
			err = p.handleDocUpdateLog(update)
		} else {
			log.Info(p.ctx, "Skipping log with invalid priority of 0", logging.NewKV("CID", update.Cid))
		}

		if err != nil {
			log.ErrorE(p.ctx, "Error while handling broadcast log", err)
		}
	}
}

// RegisterNewDocument registers a new document with the peer node.
func (p *Peer) RegisterNewDocument(
	ctx context.Context,
	dockey client.DocKey,
	c cid.Cid,
	nd ipld.Node,
	schemaID string,
) error {
	log.Debug(
		p.ctx,
		"Registering a new document for our peer node",
		logging.NewKV("DocKey", dockey.String()),
	)

	// register topic
	if err := p.server.addPubSubTopic(dockey.String(), !p.server.hasPubSubTopic(schemaID)); err != nil {
		log.ErrorE(
			p.ctx,
			"Failed to create new pubsub topic",
			err,
			logging.NewKV("DocKey", dockey.String()),
		)
		return err
	}

	// publish log
	body := &pb.PushLogRequest_Body{
		DocKey:   &pb.ProtoDocKey{DocKey: dockey},
		Cid:      &pb.ProtoCid{Cid: c},
		SchemaID: []byte(schemaID),
		Creator:  p.host.ID().String(),
		Log: &pb.Document_Log{
			Block: nd.RawData(),
		},
	}
	req := &pb.PushLogRequest{
		Body: body,
	}

	return p.server.publishLog(p.ctx, schemaID, req)
}

// SetReplicator adds a target peer node as a replication destination for documents in our DB.
func (p *Peer) SetReplicator(
	ctx context.Context,
	paddr ma.Multiaddr,
	collectionNames ...string,
) (peer.ID, error) {
	txn, err := p.db.NewTxn(ctx, true)
	if err != nil {
		return "", err
	}
	store := p.db.WithTxn(txn)

	pid, err := p.setReplicator(ctx, store, paddr, collectionNames...)
	if err != nil {
		txn.Discard(ctx)
		return "", err
	}

	return pid, txn.Commit(ctx)
}

// setReplicator adds a target peer node as a replication destination for documents in our DB.
func (p *Peer) setReplicator(
	ctx context.Context,
	store client.Store,
	paddr ma.Multiaddr,
	collectionNames ...string,
) (peer.ID, error) {
	var pid peer.ID

	// verify collections
	collections := []client.Collection{}
	schemas := []string{}
	if len(collectionNames) == 0 {
		var err error
		collections, err = store.GetAllCollections(ctx)
		if err != nil {
			return pid, errors.Wrap("failed to get all collections for replicator", err)
		}
		for _, col := range collections {
			schemas = append(schemas, col.SchemaID())
		}
	} else {
		for _, cName := range collectionNames {
			col, err := store.GetCollectionByName(ctx, cName)
			if err != nil {
				return pid, errors.Wrap("failed to get collection for replicator", err)
			}
			collections = append(collections, col)
			schemas = append(schemas, col.SchemaID())
		}
	}

	// extra peerID
	// Extract peer portion
	p2p, err := paddr.ValueForProtocol(ma.P_P2P)
	if err != nil {
		return pid, err
	}
	pid, err = peer.Decode(p2p)
	if err != nil {
		return pid, err
	}

	// make sure it's not ourselves
	if pid == p.host.ID() {
		return pid, errors.New("can't target ourselves as a replicator")
	}

	// add peer to peerstore
	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(paddr)
	if err != nil {
		return pid, errors.Wrap(fmt.Sprintf("Failed to address info from %s", paddr), err)
	}

	// Add the destination's peer multiaddress in the peerstore.
	// This will be used during connection and stream creation by libp2p.
	p.host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

	// make sure we're not duplicating things
	p.mu.Lock()
	for _, col := range collections {
		if reps, exists := p.replicators[col.SchemaID()]; exists {
			if _, exists := reps[pid]; exists {
				p.mu.Unlock()
				return pid, errors.New(fmt.Sprintf(
					"Replicator already exists for %s with PeerID %s",
					col.Name(),
					pid,
				))
			}
		} else {
			p.replicators[col.SchemaID()] = make(map[peer.ID]struct{})
		}
		// add to replicators list for the collection
		p.replicators[col.SchemaID()][pid] = struct{}{}
	}
	p.mu.Unlock()

	// Persist peer in datastore
	err = p.db.SetReplicator(ctx, client.Replicator{
		Info:    *info,
		Schemas: schemas,
	})
	if err != nil {
		return pid, errors.Wrap("failed to persist replicator", err)
	}

	for _, col := range collections {
		// create read only txn and assign to col
		txn, err := p.db.NewTxn(ctx, true)
		if err != nil {
			return pid, errors.Wrap("failed to get txn", err)
		}
		col = col.WithTxn(txn)

		// get dockeys (all)
		keysCh, err := col.GetAllDocKeys(ctx)
		if err != nil {
			txn.Discard(ctx)
			return pid, errors.Wrap(
				fmt.Sprintf(
					"Failed to get dockey for replicator %s on %s",
					pid,
					col.Name(),
				),
				err,
			)
		}

		p.pushToReplicator(ctx, txn, col, keysCh, pid)
	}
	return pid, nil
}

func (p *Peer) pushToReplicator(
	ctx context.Context,
	txn datastore.Txn,
	collection client.Collection,
	keysCh <-chan client.DocKeysResult,
	pid peer.ID,
) {
	for key := range keysCh {
		if key.Err != nil {
			log.ErrorE(ctx, "Key channel error", key.Err)
			continue
		}
		dockey := core.DataStoreKeyFromDocKey(key.Key)
		headset := clock.NewHeadSet(
			txn.Headstore(),
			dockey.WithFieldId(core.COMPOSITE_NAMESPACE).ToHeadStoreKey(),
		)
		cids, priority, err := headset.List(ctx)
		if err != nil {
			log.ErrorE(
				ctx,
				"Failed to get heads",
				err,
				logging.NewKV("DocKey", key.Key.String()),
				logging.NewKV("PeerID", pid),
				logging.NewKV("Collection", collection.Name()))
			continue
		}
		// loop over heads, get block, make the required logs, and send
		for _, c := range cids {
			blk, err := txn.DAGstore().Get(ctx, c)
			if err != nil {
				log.ErrorE(ctx, "Failed to get block", err,
					logging.NewKV("CID", c),
					logging.NewKV("PeerID", pid),
					logging.NewKV("Collection", collection.Name()))
				continue
			}

			// @todo: remove encode/decode loop for core.Log data
			nd, err := dag.DecodeProtobuf(blk.RawData())
			if err != nil {
				log.ErrorE(ctx, "Failed to decode protobuf", err, logging.NewKV("CID", c))
				continue
			}

			evt := events.Update{
				DocKey:   key.Key.String(),
				Cid:      c,
				SchemaID: collection.SchemaID(),
				Block:    nd,
				Priority: priority,
			}
			if err := p.server.pushLog(ctx, evt, pid); err != nil {
				log.ErrorE(
					ctx,
					"Failed to replicate log",
					err,
					logging.NewKV("CID", c),
					logging.NewKV("PeerID", pid),
				)
			}
		}
	}
}

// DeleteReplicator adds a target peer node as a replication destination for documents in our DB.
func (p *Peer) DeleteReplicator(
	ctx context.Context,
	pid peer.ID,
	collectionNames ...string,
) error {
	txn, err := p.db.NewTxn(ctx, true)
	if err != nil {
		return err
	}
	store := p.db.WithTxn(txn)

	err = p.deleteReplicator(ctx, store, pid, collectionNames...)
	if err != nil {
		txn.Discard(ctx)
		return err
	}

	return txn.Commit(ctx)
}

// DeleteReplicator adds a target peer node as a replication destination for documents in our DB.
func (p *Peer) deleteReplicator(
	ctx context.Context,
	store client.Store,
	pid peer.ID,
	collectionNames ...string,
) error {
	// make sure it's not ourselves
	if pid == p.host.ID() {
		return errors.New("can't target ourselves as a replicator")
	}

	// verify collections
	schemas := []string{}
	schemaMap := make(map[string]struct{})
	if len(collectionNames) == 0 {
		var err error
		collections, err := store.GetAllCollections(ctx)
		if err != nil {
			return errors.Wrap("failed to get all collections for replicator", err)
		}
		for _, col := range collections {
			schemas = append(schemas, col.SchemaID())
			schemaMap[col.SchemaID()] = struct{}{}
		}
	} else {
		for _, cName := range collectionNames {
			col, err := store.GetCollectionByName(ctx, cName)
			if err != nil {
				return errors.Wrap("failed to get collection for replicator", err)
			}
			schemas = append(schemas, col.SchemaID())
			schemaMap[col.SchemaID()] = struct{}{}
		}
	}

	// make sure we're not duplicating things
	p.mu.Lock()
	defer p.mu.Unlock()

	totalSchemas := 0 // Lets keep track of how many schemas are left for the replicator.
	for schema, rep := range p.replicators {
		if _, exists := rep[pid]; exists {
			if _, toDelete := schemaMap[schema]; toDelete {
				delete(p.replicators[schema], pid)
			} else {
				totalSchemas++
			}
		}
	}

	if totalSchemas == 0 {
		// Remove the destination's peer multiaddress in the peerstore.
		p.host.Peerstore().ClearAddrs(pid)
	}

	// Delete peer in datastore
	return p.db.DeleteReplicator(ctx, client.Replicator{
		Info:    peer.AddrInfo{ID: pid},
		Schemas: schemas,
	})
}

// GetAllReplicators adds a target peer node as a replication destination for documents in our DB.
func (p *Peer) GetAllReplicators(ctx context.Context) ([]client.Replicator, error) {
	return p.db.GetAllReplicators(ctx)
}

func (p *Peer) loadReplicators(ctx context.Context) error {
	reps, err := p.db.GetAllReplicators(ctx)
	if err != nil {
		return errors.Wrap("failed to get replicators", err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, rep := range reps {
		for _, schema := range rep.Schemas {
			if pReps, exists := p.replicators[schema]; exists {
				if _, exists := pReps[rep.Info.ID]; exists {
					continue
				}
			} else {
				p.replicators[schema] = make(map[peer.ID]struct{})
			}

			// add to replicators list
			p.replicators[schema][rep.Info.ID] = struct{}{}
		}

		// Add the destination's peer multiaddress in the peerstore.
		// This will be used during connection and stream creation by libp2p.
		p.host.Peerstore().AddAddrs(rep.Info.ID, rep.Info.Addrs, peerstore.PermanentAddrTTL)

		log.Info(ctx, "loaded replicators from datastore", logging.NewKV("Replicator", rep))
	}

	return nil
}

func (p *Peer) loadP2PCollections(ctx context.Context) (map[string]struct{}, error) {
	collections, err := p.db.GetAllP2PCollections(ctx)
	if err != nil && !errors.Is(err, ds.ErrNotFound) {
		return nil, err
	}
	colMap := make(map[string]struct{})
	for _, col := range collections {
		err := p.server.addPubSubTopic(col, true)
		if err != nil {
			return nil, err
		}
		colMap[col] = struct{}{}
	}

	return colMap, nil
}

func (p *Peer) handleDocCreateLog(evt events.Update) error {
	dockey, err := client.NewDocKeyFromString(evt.DocKey)
	if err != nil {
		return errors.Wrap("failed to get DocKey from broadcast message", err)
	}

	// We need to register the document before pushing to the replicators if we want to
	// ensure that we have subscribed to the topic.
	err = p.RegisterNewDocument(p.ctx, dockey, evt.Cid, evt.Block, evt.SchemaID)
	if err != nil {
		return err
	}
	// push to each peer (replicator)
	p.pushLogToReplicators(p.ctx, evt)

	return nil
}

func (p *Peer) handleDocUpdateLog(evt events.Update) error {
	dockey, err := client.NewDocKeyFromString(evt.DocKey)
	if err != nil {
		return errors.Wrap("failed to get DocKey from broadcast message", err)
	}
	log.Debug(
		p.ctx,
		"Preparing pubsub pushLog request from broadcast",
		logging.NewKV("DocKey", dockey),
		logging.NewKV("CID", evt.Cid),
		logging.NewKV("SchemaId", evt.SchemaID))

	body := &pb.PushLogRequest_Body{
		DocKey:   &pb.ProtoDocKey{DocKey: dockey},
		Cid:      &pb.ProtoCid{Cid: evt.Cid},
		SchemaID: []byte(evt.SchemaID),
		Creator:  p.host.ID().String(),
		Log: &pb.Document_Log{
			Block: evt.Block.RawData(),
		},
	}
	req := &pb.PushLogRequest{
		Body: body,
	}

	// push to each peer (replicator)
	p.pushLogToReplicators(p.ctx, evt)

	if err := p.server.publishLog(p.ctx, evt.DocKey, req); err != nil {
		return errors.Wrap(fmt.Sprintf("can't publish log %s for dockey %s", evt.Cid, evt.DocKey), err)
	}

	if err := p.server.publishLog(p.ctx, evt.SchemaID, req); err != nil {
		return errors.Wrap(fmt.Sprintf("can't publish log %s for schemaID %s", evt.Cid, evt.SchemaID), err)
	}

	return nil
}

func (p *Peer) pushLogToReplicators(ctx context.Context, lg events.Update) {
	// push to each peer (replicator)
	peers := make(map[string]struct{})
	for _, peer := range p.ps.ListPeers(lg.DocKey) {
		peers[peer.String()] = struct{}{}
	}
	for _, peer := range p.ps.ListPeers(lg.SchemaID) {
		peers[peer.String()] = struct{}{}
	}

	p.mu.Lock()
	reps, exists := p.replicators[lg.SchemaID]
	p.mu.Unlock()

	if exists {
		for pid := range reps {
			// Don't push if pid is in the list of peers for the topic.
			// It will be handled by the pubsub system.
			if _, ok := peers[pid.String()]; ok {
				continue
			}
			go func(peerID peer.ID) {
				if err := p.server.pushLog(p.ctx, lg, peerID); err != nil {
					log.ErrorE(
						p.ctx,
						"Failed pushing log",
						err,
						logging.NewKV("DocKey", lg.DocKey),
						logging.NewKV("CID", lg.Cid),
						logging.NewKV("PeerID", peerID))
				}
			}(pid)
		}
	}
}

func (p *Peer) setupBlockService() {
	bswapnet := network.NewFromIpfsHost(p.host, p.dht)
	bswap := bitswap.New(p.ctx, bswapnet, p.db.Blockstore())
	p.bserv = blockservice.New(p.db.Blockstore(), bswap)
	p.exch = bswap
}

func (p *Peer) setupDAGService() {
	p.DAGService = dag.NewDAGService(p.bserv)
}

func (p *Peer) newDAGSyncerTxn(txn datastore.Txn) ipld.DAGService {
	return dag.NewDAGService(blockservice.New(txn.DAGstore(), p.exch))
}

// Session returns a session-based NodeGetter.
func (p *Peer) Session(ctx context.Context) ipld.NodeGetter {
	ng := dag.NewSession(ctx, p.DAGService)
	if ng == p.DAGService {
		log.Info(ctx, "DAGService does not support sessions")
	}
	return ng
}

func stopGRPCServer(ctx context.Context, server *grpc.Server) {
	stopped := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(stopped)
	}()
	timer := time.NewTimer(10 * time.Second)
	select {
	case <-timer.C:
		server.Stop()
		log.Info(ctx, "Peer gRPC server was shutdown ungracefully")
	case <-stopped:
		timer.Stop()
	}
}

type EvtReceivedPushLog struct {
	ByPeer   peer.ID
	FromPeer peer.ID
}

type EvtPubSub struct {
	Peer peer.ID
}

// AddP2PCollections adds the given collectionIDs to the pubsup topics.
//
// It will error if any of the given collectionIDs are invalid, in such a case some of the
// changes to the server may still be applied.
//
// WARNING: Calling this on collections with a large number of documents may take a long time to process.
func (p *Peer) AddP2PCollections(collections []string) error {
	txn, err := p.db.NewTxn(p.ctx, false)
	if err != nil {
		return err
	}
	defer txn.Discard(p.ctx)
	store := p.db.WithTxn(txn)

	// first let's make sure the collections actually exists
	for _, col := range collections {
		_, err := store.GetCollectionBySchemaID(p.ctx, col)
		if err != nil {
			return err
		}
	}

	// Ensure we can add all the collections to the store on the transaction
	// before adding to topics.
	for _, col := range collections {
		err := store.AddP2PCollection(p.ctx, col)
		if err != nil {
			return err
		}
	}

	// Add pubsub topics and remove them if we get an error.
	addedTopics := []string{}
	for _, col := range collections {
		err = p.server.addPubSubTopic(col, true)
		if err != nil {
			for _, topic := range addedTopics {
				e := p.server.removePubSubTopic(topic)
				if e != nil {
					return errors.WithStack(e, errors.NewKV("Cause", err))
				}
			}
			return err
		}
		addedTopics = append(addedTopics, col)
	}

	// If adding the collection topics succeeds, we remove the collections' documents
	// from the pubsub topics to avoid receiving duplicate events.
	for _, col := range collections {
		c, err := store.GetCollectionBySchemaID(p.ctx, col)
		if err != nil {
			return err
		}
		keyChan, err := c.GetAllDocKeys(p.ctx)
		if err != nil {
			return err
		}
		for key := range keyChan {
			err := p.server.removePubSubTopic(key.Key.String())
			if err != nil {
				log.Info(
					p.ctx,
					"Failed to remove doc from pubsub topic",
					logging.NewKV("DocKey", key.Key.String()),
					logging.NewKV("Cause", err),
				)
			}
		}
	}

	return txn.Commit(p.ctx)
}

// RemoveP2PCollections removes the given collectionIDs from the pubsup topics.
//
// It will error if any of the given collectionIDs are invalid, in such a case some of the
// changes to the server may still be applied.
//
// WARNING: Calling this on collections with a large number of documents may take a long time to process.
func (p *Peer) RemoveP2PCollections(collections []string) error {
	txn, err := p.db.NewTxn(p.ctx, false)
	if err != nil {
		return err
	}
	defer txn.Discard(p.ctx)
	store := p.db.WithTxn(txn)

	// first let's make sure the collections actually exists
	for _, col := range collections {
		_, err := store.GetCollectionBySchemaID(p.ctx, col)
		if err != nil {
			return err
		}
	}

	// Ensure we can remove all the collections to the store on the transaction
	// before adding to topics.
	for _, col := range collections {
		err := store.RemoveP2PCollection(p.ctx, col)
		if err != nil {
			return err
		}
	}

	// Remove pubsub topics and add them back if we get an error.
	removedTopics := []string{}
	for _, col := range collections {
		err = p.server.removePubSubTopic(col)
		if err != nil {
			for _, topic := range removedTopics {
				e := p.server.addPubSubTopic(topic, true)
				if e != nil {
					return errors.WithStack(e, errors.NewKV("Cause", err))
				}
			}
			return err
		}
	}

	// If removing the collection topics succeeds, we add back the collections' documents
	// to the pubsub topics.
	for _, col := range collections {
		c, err := store.GetCollectionBySchemaID(p.ctx, col)
		if err != nil {
			return err
		}
		keyChan, err := c.GetAllDocKeys(p.ctx)
		if err != nil {
			return err
		}
		for key := range keyChan {
			err := p.server.addPubSubTopic(key.Key.String(), true)
			if err != nil {
				log.Info(
					p.ctx,
					"Failed to add doc to pubsub topic",
					logging.NewKV("DocKey", key.Key.String()),
					logging.NewKV("Cause", err),
				)
			}
		}
	}

	return txn.Commit(p.ctx)
}

// GetAllP2PCollections gets all the collectionIDs from the pubsup topics
func (p *Peer) GetAllP2PCollections() ([]client.P2PCollection, error) {
	txn, err := p.db.NewTxn(p.ctx, false)
	if err != nil {
		return nil, err
	}
	store := p.db.WithTxn(txn)

	collections, err := p.db.GetAllP2PCollections(p.ctx)
	if err != nil {
		txn.Discard(p.ctx)
		return nil, err
	}

	p2pCols := []client.P2PCollection{}
	for _, colID := range collections {
		col, err := store.GetCollectionBySchemaID(p.ctx, colID)
		if err != nil {
			txn.Discard(p.ctx)
			return nil, err
		}
		p2pCols = append(p2pCols, client.P2PCollection{
			ID:   colID,
			Name: col.Name(),
		})
	}

	return p2pCols, txn.Commit(p.ctx)
}

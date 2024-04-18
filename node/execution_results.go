package node

import (
	"context"
	"github.com/allora-network/b7s/models/response"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"

	"github.com/allora-network/b7s/models/execute"
)

// gatherExecutionResultsPBFT collects execution results from a PBFT cluster. This means f+1 identical results.
func (n *Node) gatherExecutionResultsConsensus(requestID string, peers []peer.ID) execute.ResultMap {

	var (
		lock sync.Mutex
		wg   sync.WaitGroup

		out execute.ResultMap = make(map[peer.ID]execute.Result)
	)

	wg.Add(len(peers))

	for _, rp := range peers {
		go func(sender peer.ID) {
			defer wg.Done()

			key := executionResultKey(requestID, sender)
			res, ok := n.consensusExecuteResponse[requestID][key]
			if !ok {
				return
			}

			n.log.Info().Str("peer", sender.String()).Str("request", requestID).Msg("accounted execution response from peer")

			if len(res.Signature) > 0 {
				pub, err := sender.ExtractPublicKey()
				if err != nil {
					log.Error().Err(err).Msg("could not derive public key from peer ID")
					return
				}

				err = res.VerifySignature(pub)
				if err != nil {
					log.Error().Err(err).Msg("could not verify signature of an execution response")
					return
				}
			}

			exres, ok := res.Results[sender]
			if !ok {
				return
			}

			lock.Lock()
			out[sender] = exres
			defer lock.Unlock()
		}(rp)
	}

	wg.Wait()

	return out
}

// gatherExecutionResults collects execution results from direct executions or raft clusters.
func (n *Node) gatherExecutionResults(ctx context.Context, requestID string, peers []peer.ID) execute.ResultMap {
	// We're willing to wait for a limited amount of time.
	exctx, exCancel := context.WithTimeout(ctx, n.cfg.ExecutionTimeout)
	defer exCancel()

	var (
		results execute.ResultMap = make(map[peer.ID]execute.Result)
		reslock sync.Mutex
		wg      sync.WaitGroup
	)

	wg.Add(len(peers))

	// Wait on peers asynchronously.
	for _, rp := range peers {
		rp := rp

		go func() {
			defer wg.Done()
			key := executionResultKey(requestID, rp)
			res, ok := n.executeResponses.WaitFor(exctx, key)

			if !ok {
				return
			}

			n.log.Info().Str("peer", rp.String()).Msg("accounted execution response from peer")

			er := res.(response.Execute)

			exres, ok := er.Results[rp]
			if !ok {
				return
			}

			reslock.Lock()
			defer reslock.Unlock()
			results[rp] = exres
		}()
	}

	wg.Wait()

	return results
}

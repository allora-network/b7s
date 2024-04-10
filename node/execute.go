package node

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/allora-network/b7s/consensus"
	"github.com/allora-network/b7s/models/codes"
	"github.com/allora-network/b7s/models/execute"
	"github.com/allora-network/b7s/models/response"
)

func (n *Node) processExecute(ctx context.Context, from peer.ID, payload []byte) error {
	// We execute functions differently depending on the node role.
	if n.isHead() {
		return n.headProcessExecute(ctx, from, payload)
	}
	return n.workerProcessExecute(ctx, from, payload)
}

func (n *Node) processExecuteResponse(ctx context.Context, from peer.ID, payload []byte) error {

	// Unpack the message.
	var res response.Execute
	err := json.Unmarshal(payload, &res)
	if err != nil {
		return fmt.Errorf("could not unpack execute response: %w", err)
	}
	res.From = from

	n.log.Debug().Str("request", res.RequestID).Str("from", from.String()).Msg("received execution response")

	key := executionResultKey(res.RequestID, from)
	n.executeResponses.Set(key, res)

	return nil
}

func (n *Node) processExecuteResponseToPrimary(ctx context.Context, from peer.ID, payload []byte) error {

	// Unpack the message.
	var res response.Execute
	err := json.Unmarshal(payload, &res)
	if err != nil {
		return fmt.Errorf("could not unpack execute response: %w", err)
	}
	res.From = from
	key := executionResultKey(res.RequestID, from)
	n.pbftExecuteResponseLock.Lock()
	n.pbftExecuteResponse[key] = res
	n.pbftExecuteResponseLock.Unlock()
	if len(n.reportingPeers[res.RequestID]) > 0 && len(n.pbftExecuteResponse) >= len(n.reportingPeers[res.RequestID])-1 {
		out := n.gatherExecutionResultsPBFT(res.RequestID, n.reportingPeers[res.RequestID])
		result := codes.OK
		if len(out) == 0 {
			result = codes.Error
		}
		send := &ChanData{
			Res:        result,
			FunctionId: res.FunctionID,
			RequestId:  res.RequestID,
			Topic:      n.topics[res.RequestID],
			Data:       out,
		}
		payload, err := json.Marshal(send)
		if err != nil {
			fmt.Errorf("could not pack execute response for sending application layer: %w", err)
		}
		n.comChannel <- payload
		_ = n.disbandCluster(res.RequestID, n.reportingPeers[res.RequestID])
	}

	return nil
}

func (n *Node) listenClusterChannel(ctx context.Context) {
	select {
	case msg := <-n.clusterChannel:
		_ = n.processExecuteResponseToPrimary(nil, n.host.ID(), msg)
	}
}
func executionResultKey(requestID string, peer peer.ID) string {
	return requestID + "/" + peer.String()
}

// determineOverallCode will return the resulting code from a set of results. Rules are:
// - if there's a single result, we use that results code
// - return OK if at least one result was successful
// - return error if none of the results were successful
func determineOverallCode(results map[string]execute.Result) codes.Code {

	if len(results) == 0 {
		return codes.NoContent
	}

	// For a single peer, just return its code.
	if len(results) == 1 {
		for peer := range results {
			return results[peer].Code
		}
	}

	// For multiple results - return OK if any of them succeeded.
	for _, res := range results {
		if res.Code == codes.OK {
			return codes.OK
		}
	}

	return codes.Error
}

func parseConsensusAlgorithm(value string) (consensus.Type, error) {

	if value == "" {
		return 0, nil
	}

	lv := strings.ToLower(value)
	switch lv {
	case "raft":
		return consensus.Raft, nil

	case "pbft":
		return consensus.PBFT, nil
	}

	return 0, fmt.Errorf("unknown consensus value (%s)", value)
}

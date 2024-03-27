package pbft

import (
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/RedBird96/b7s/models/blockless"
	"github.com/RedBird96/b7s/models/codes"
	"github.com/RedBird96/b7s/models/execute"
	"github.com/RedBird96/b7s/models/response"
)

// Execute fullfils the consensus interface by inserting the request into the pipeline.
func (r *Replica) Execute(client peer.ID, requestID string, timestamp time.Time, req execute.Request) (codes.Code, execute.Result, error) {

	// Modifying state, so acquire state lock now.
	r.sl.Lock()
	defer r.sl.Unlock()

	request := Request{
		ID:        requestID,
		Timestamp: timestamp,
		Origin:    client,
		Execute:   req,
	}

	err := r.processRequest(client, request)
	if err != nil {
		return codes.Error, execute.Result{}, fmt.Errorf("could not process request: %w", err)
	}

	// Nothing to return at this point.
	return codes.NoContent, execute.Result{}, nil
}

// execute executes the request AND sends the result back to origin.
func (r *Replica) execute(view uint, sequence uint, digest string) error {

	// Sanity check, should not happen.
	request, ok := r.requests[digest]
	if !ok {
		return fmt.Errorf("unknown request (digest: %s)", digest)
	}

	log := r.log.With().Uint("view", view).Uint("sequence", sequence).Str("digest", digest).Str("request", request.ID).Logger()

	// We don't want to execute a job multiple times.
	_, havePending := r.pending[digest]
	if !havePending {
		log.Warn().Msg("no pending request with matching info - likely already executed")
		return nil
	}

	// Requests must be executed in order.
	if sequence != r.lastExecuted+1 {
		log.Error().Msg("requests with lower sequence number have not been executed")
		// TODO (pbft): Start execution of earlier requests?
		return nil
	}

	// Sanity check - should never happen.
	if sequence < r.lastExecuted {
		log.Error().Uint("last_executed", r.lastExecuted).Msg("requests executed out of order!")
	}

	// Remove this request from the list of outstanding requests.
	delete(r.pending, digest)

	log.Info().Msg("executing request")

	res, err := r.executor.ExecuteFunction(request.ID, request.Execute)
	if err != nil {
		log.Error().Err(err).Msg("execution failed")
	}

	// Stop the timer since we completed an execution.
	r.stopRequestTimer()

	// If we have more pending requests, start a new timer.
	if len(r.pending) > 0 {
		r.startRequestTimer(true)
	}

	log.Info().Msg("executed request")

	r.lastExecuted = sequence

	msg := response.Execute{
		Type:       blockless.MessageExecuteResponseToPrimary,
		Code:       res.Code,
		RequestID:  request.ID,
		FunctionID: request.Execute.FunctionID,
		Results: execute.ResultMap{
			r.id: res,
		},
		PBFT: response.PBFTResultInfo{
			View:             r.view,
			RequestTimestamp: request.Timestamp,
			Replica:          r.id,
		},
	}

	// Save this executions in case it's requested again.
	r.executions[request.ID] = msg

	// Invoke specified post processor functions.
	for _, proc := range r.cfg.PostProcessors {
		proc(request.ID, request.Origin, request.Execute, res)
	}

	err = msg.Sign(r.host.PrivateKey())
	if err != nil {
		return fmt.Errorf("could not sign execution request: %w", err)
	}
	if r.host.ID() == r.primaryReplicaID() {
		return nil
	}

	log.Log().Msgf("Primary Peer %s", r.primaryReplicaID().String())
	err = r.send(r.primaryReplicaID(), msg, blockless.ProtocolID)
	if err != nil {
		return fmt.Errorf("could not send execution response to node (current: %s, target: %s, request: %s): %w", r.host.ID(), r.primaryReplicaID(), request.ID, err)
	}

	return nil
}

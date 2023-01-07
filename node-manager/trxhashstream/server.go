// Copyright 2021 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package trxhashstream

import (
	"sync"

	"github.com/streamingfast/logging"
	pbtrxhash "github.com/streamingfast/firehose-ethereum/types/pb/sf/ethereum/indigo"
	"github.com/streamingfast/shutter"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	*shutter.Shutter

	subscriptions []*subscription
	lock          sync.RWMutex

	logger *zap.Logger
}

func NewServer(logger *zap.Logger) *Server {
	return &Server{
		Shutter: shutter.New(),
		logger:  logger,
	}
}

func (s *Server) TransactionHashes(r *pbtrxhash.HashRequest, stream pbtrxhash.TransactionHashStream_HashesServer) error {
	zlogger := logging.Logger(stream.Context(), s.logger)

	subscription := s.subscribe(zlogger)
	defer s.unsubscribe(subscription)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-s.Terminating():
			return status.Error(codes.Unavailable, "service is terminating")
		case trxHash, ok := <-subscription.incomingTrxHash:
			if !ok {
				// we've been shutdown somehow, simply close the current connection..
				// we'll have logged at the source
				return nil
			}
			zlogger.Debug("sending hash to subscription", zap.Stringer("hash", trxHash))
			err := stream.Send(trxHash)
			if err != nil {
				zlogger.Info("failed writing to socket, shutting down subscription", zap.Error(err))
				break
			}
		}
	}
}

func (s *Server) Ready() bool {
	return true
}

func (s *Server) PushTransactionHash(trxHash *pbtrxhash.TransactionHash) {
	if s.IsTerminating() {
		return
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	for _, sub := range s.subscriptions {
		sub.Push(trxHash)
	}
}

func (s *Server) subscribe(logger *zap.Logger) *subscription {
	chanSize := 200
	sub := newSubscription(chanSize, s.logger.Named("sub"))

	s.lock.Lock()
	defer s.lock.Unlock()

	s.subscriptions = append(s.subscriptions, sub)
	s.logger.Info("subscribed", zap.Int("new_length", len(s.subscriptions)))

	return sub
}

func (s *Server) unsubscribe(toRemove *subscription) {
	var newListeners []*subscription
	for _, sub := range s.subscriptions {
		if sub != toRemove {
			newListeners = append(newListeners, sub)
		}
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.subscriptions = newListeners
	s.logger.Info("unsubscribed", zap.Int("new_length", len(s.subscriptions)))
}

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

package nodemanager

import (
	"fmt"
	"strings"

	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/firehose-ethereum/codec"
	"github.com/streamingfast/firehose-ethereum/node-manager/trxhashstream"
	pbtrxhash "github.com/streamingfast/firehose-ethereum/types/pb/sf/ethereum/indigo"
	"github.com/streamingfast/shutter"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type TrxHashLogPlugin struct {
	*shutter.Shutter

	logLines chan string
	server   *trxhashstream.Server
	logger   *zap.Logger
}

func NewTrxHashLogPlugin(logger *zap.Logger) *TrxHashLogPlugin {
	trxServer := trxhashstream.NewServer(logger)

	return &TrxHashLogPlugin{
		Shutter: shutter.New(),

		server:   trxServer,
		logLines: make(chan string),
		logger:   logger,
	}
}

func (p *TrxHashLogPlugin) Launch() {}
func (p TrxHashLogPlugin) Stop()    {}

func (p *TrxHashLogPlugin) Name() string {
	return "TrxHashLogPlugin"
}
func (p *TrxHashLogPlugin) Close(_ error) {
	p.server.Shutdown(nil)
}

func (p *TrxHashLogPlugin) RegisterServices(gs grpc.ServiceRegistrar) {
	pbtrxhash.RegisterTransactionStreamServer(gs, p.server)
}

func (p *TrxHashLogPlugin) LogLine(line string) {
	switch {
	case strings.HasPrefix(line, "FIRE TRX_HASH_BROADCAST"):
		line = line[5:]
	default:
		return
	}

	p.logger.Debug("detected trx hash broadcast event detected")
	chunks, err := codec.SplitInChunks(line, 5)
	if err != nil {
		panic(fmt.Errorf("failed to spit log line %q: %w", line, err))
	}

	tx := readTrxHashBegin(chunks)
	p.logger.Debug("pushing transaction", zap.Stringer("trx_id", eth.Hash(tx.Hash)))
	p.server.PushTransactionHash(tx)
}

func readTrxHashBegin(chunks []string) *pbtrxhash.TransactionHash {
	hash := codec.FromHex(chunks[0], "TRX_HASH txHash")
	peer_id := codec.From(chunks[1])
	batch_size := codec.FromUint64(chunks[2], "TRX_HASH batchSize")
	received_at := codec.FromUint64(codec.FromHex(chunks[3], "TRX_HASH receivedAt"))

	return &pbtrxhash.TransactionHash{
		Hash:     		hash,
		PeerId:   		peer_id,
		BatchSize:      batch_size,
		ReceivedAt:     received_at,
	}
}

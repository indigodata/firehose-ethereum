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

package codec

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/streamingfast/firehose-ethereum/types"
	pbeth "github.com/streamingfast/firehose-ethereum/types/pb/sf/ethereum/type/v2"
	"github.com/streamingfast/jsonpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFromFile(t *testing.T) {
	tests := []struct {
		deepMindFile     string
		expectedErr      error
		expectedPanicErr error
		readTransaction  bool
	}{
		{"testdata/firehose-logs.dmlog", nil, nil, false},
		{"testdata/normalize-r-and-s-curve-points.dmlog", nil, nil, false},
		{"testdata/block_mining_rewards.dmlog", nil, nil, false},
		{"testdata/block_unknown_balance_change.dmlog", nil, errors.New(`receive unknown balance change reason, received reason string is "something_that_will_never_match"`), false},
		{"testdata/read_transaction.dmlog", nil, nil, true},
		{"testdata/read_transaction_access_list.dmlog", nil, nil, true},
		{"testdata/read_transaction_dynamic_fee.dmlog", nil, nil, true},
		{"testdata/polygon_calls_after_finalize.dmlog", nil, nil, false},
		{"testdata/polygon_add_log_0.dmlog", nil, nil, false},
		{"testdata/lachesis.dmlog", nil, nil, false},
	}

	for _, test := range tests {
		t.Run(strings.Replace(test.deepMindFile, "testdata/", "", 1), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					require.Equal(t, test.expectedPanicErr, r, "Panicked with %s", r)
				}
			}()

			cr := testFileConsoleReader(t, test.deepMindFile)

			var reader ObjectReader = func() (interface{}, error) {
				out, err := cr.ReadBlock()
				if err != nil {
					return nil, err
				}

				return out.ToProtocol().(*pbeth.Block), nil
			}

			if test.readTransaction {
				reader = func() (interface{}, error) {
					return cr.ReadTransaction()
				}
			}

			buf := &bytes.Buffer{}
			buf.Write([]byte("["))
			first := true

			for {
				out, err := reader()
				if v, ok := out.(proto.Message); ok && !isNil(v) {
					if !first {
						buf.Write([]byte(","))
					}
					first = false

					value, err := jsonpb.MarshalIndentToString(v, "  ")
					require.NoError(t, err)

					buf.Write([]byte(value))
				}

				if err == io.EOF {
					break
				}

				if len(buf.Bytes()) != 0 {
					buf.Write([]byte("\n"))
				}

				if test.expectedErr == nil {
					require.NoError(t, err)
				} else if err != nil {
					require.Equal(t, test.expectedErr, err)
					return
				}
			}
			buf.Write([]byte("]"))

			goldenFile := test.deepMindFile + ".golden.json"
			if os.Getenv("GOLDEN_UPDATE") == "true" {
				ioutil.WriteFile(goldenFile, buf.Bytes(), os.ModePerm)
			}

			cnt, err := ioutil.ReadFile(goldenFile)
			require.NoError(t, err)

			if !assert.JSONEq(t, string(cnt), buf.String()) {
				t.Error("previous diff:\n" + unifiedDiff(t, cnt, buf.Bytes()))
			}
		})
	}
}

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Ptr && rv.IsNil()
}

func TestGeneratePBBlocks(t *testing.T) {
	t.Skip("generate only when firehose-logs.dmlog changes")

	cr := testFileConsoleReader(t, "testdata/firehose-logs.dmlog")

	for {
		out, err := cr.ReadBlock()
		if out != nil {
			block := out.ToProtocol().(*pbeth.Block)

			bstreamBlock, err := types.BlockFromProto(block, out.LibNum)
			require.NoError(t, err)

			pbBlock, err := bstreamBlock.ToProto()
			require.NoError(t, err)

			outputFile, err := os.Create(fmt.Sprintf("testdata/pbblocks/battlefield-block.%d.pb", block.Number))
			require.NoError(t, err)

			bytes, err := proto.Marshal(pbBlock)
			require.NoError(t, err)

			_, err = outputFile.Write(bytes)
			require.NoError(t, err)

			outputFile.Close()
		}

		if err == io.EOF {
			break
		}

		require.NoError(t, err)
	}
}

func consumeBlock(t *testing.T, reader *ConsoleReader) *pbeth.Block {
	t.Helper()

	block, err := reader.ReadBlock()
	if block == nil {
		require.Fail(t, err.Error())
	}

	return block.ToProtocol().(*pbeth.Block)
}

func consumeSingleBlock(t *testing.T, reader *ConsoleReader) *pbeth.Block {
	t.Helper()

	block := consumeBlock(t, reader)
	consumeToEOF(t, reader)

	return block
}

func consumeToEOF(t *testing.T, reader *ConsoleReader) {
	block, err := reader.ReadBlock()
	require.Nil(t, block)
	require.Equal(t, err, io.EOF)

	return
}

func testFileConsoleReader(t *testing.T, filename string) *ConsoleReader {
	t.Helper()

	fl, err := os.Open(filename)
	require.NoError(t, err)

	cr := testReaderConsoleReader(t.Helper, make(chan string, 10000), func() { fl.Close() })

	go cr.ProcessData(fl)

	return cr
}

//func testStringConsoleReader(t *testing.T, data string) *ConsoleReader {
//	t.Helper()
//
//	return testReaderConsoleReader(t, bytes.NewBufferString(data), func() {})
//}

func testReaderConsoleReader(helperFunc func(), lines chan string, closer func()) *ConsoleReader {
	l := &ConsoleReader{
		lines:  lines,
		close:  closer,
		ctx:    &parseCtx{logger: zlog, stats: newParsingStats(zlog, 0), globalStats: newConsoleReaderStats()},
		logger: zlog,
	}

	return l
}

func TestValueParsing(t *testing.T) {
	testValue := "deff"
	expectedValue := &pbeth.BigInt{
		Bytes: big.NewInt(int64(57087)).Bytes(),
	}
	value := pbeth.BigIntFromBytes(FromHex(testValue, "TESTING value"))
	require.Equal(t, expectedValue, value)

}

func bytesListToHexList(bytesList [][]byte) []string {
	hexes := make([]string, len(bytesList))
	for i, bytes := range bytesList {
		hexes[i] = hex.EncodeToString(bytes)
	}

	return hexes
}

func unifiedDiff(t *testing.T, cnt1, cnt2 []byte) string {
	file1 := "/tmp/gotests-linediff-1"
	file2 := "/tmp/gotests-linediff-2"
	err := ioutil.WriteFile(file1, cnt1, 0600)
	require.NoError(t, err)

	err = ioutil.WriteFile(file2, cnt2, 0600)
	require.NoError(t, err)

	cmd := exec.Command("diff", "-u", file1, file2)
	out, _ := cmd.Output()

	return string(out)
}

func Test_computeProofOfWorkLIBNum(t *testing.T) {
	type args struct {
		blockNum                uint64
		firstStreamableBlockNum uint64
	}

	tests := []struct {
		name string
		args args
		want uint64
	}{
		{"block is before first streamable block", args{0, 200}, 200},
		{"block is equal to first streamable block", args{200, 200}, 200},
		{"block is after first streamable block", args{201, 200}, 200},
		{"block is direct +200 blocks from first streamable block", args{400, 200}, 200},
		{"block is direct +201 blocks from first streamable block", args{401, 200}, 201},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, computeProofOfWorkLIBNum(tt.args.blockNum, tt.args.firstStreamableBlockNum))
		})
	}
}

func Test_computeProofOfStakeLIBNum(t *testing.T) {
	type args struct {
		current         uint64
		finalized       uint64
		firstStreamable uint64
	}

	tests := []struct {
		name string
		args args
		want uint64
	}{
		{"current is below first streamable, finalized block below current", args{current: 10, finalized: 0, firstStreamable: 200}, 200},
		{"current is equal to first streamable, finalized block below current", args{current: 200, finalized: 0, firstStreamable: 200}, 200},

		{"current is below first streamable, finalized block above current", args{current: 10, finalized: 400, firstStreamable: 200}, 200},
		{"current is equal to first streamable, finalized block above current", args{current: 200, finalized: 400, firstStreamable: 200}, 200},

		{"current is below first streamable, finalized block below first streamable", args{current: 10, finalized: 100, firstStreamable: 200}, 200},
		{"current is equal to first streamable, finalized block below first streamable", args{current: 200, finalized: 100, firstStreamable: 200}, 200},

		{"current is below first streamable, finalized block above first streamable", args{current: 10, finalized: 400, firstStreamable: 200}, 200},
		{"current is equal to first streamable, finalized block above first streamable", args{current: 200, finalized: 400, firstStreamable: 200}, 200},

		{"current is below finalized, above first streamable", args{current: 10, finalized: 400}, 10},
		{"current is equal to finalized, above first streamable", args{current: 400, finalized: 400}, 400},
		{"current is above finalized, above first streamable", args{current: 410, finalized: 400}, 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, computeProofOfStakeLIBNum(tt.args.current, tt.args.finalized, tt.args.firstStreamable))
		})
	}
}

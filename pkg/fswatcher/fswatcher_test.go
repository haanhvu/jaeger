// Copyright (c) 2021 The Jaeger Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fswatcher

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestWatchFiles(t *testing.T) {
	w, err := NewFSWatcher()
	require.NoError(t, err)
	assert.IsType(t, &fsWatcherWrapper{}, w)

	err = w.WatchFiles([]string{"foo"}, nil, nil)
	assert.Error(t, err)

	err = w.WatchFiles([]string{"../../cmd/query/app/fixture/ui-config.json"}, nil, nil)
	assert.NoError(t, err)

	err = w.WatchFiles([]string{"../../cmd/query/app/fixture/ui-config.json", "foo"}, nil, nil)
	assert.Error(t, err)

	err = w.WatchFiles([]string{"../../cmd/query/app/fixture/static/asset.txt", "../../cmd/query/app/fixture/ui-config.json"}, nil, nil)
	assert.NoError(t, err)

	err = w.Close()
	assert.NoError(t, err)
}

func TestWatchFilesChangeAndRemove(t *testing.T) {
	w, err := NewFSWatcher()
	require.NoError(t, err)
	assert.IsType(t, &fsWatcherWrapper{}, w)

	testFile, err := os.Create("test.doc")
	require.NoError(t, err)

	_, err = testFile.WriteString("test content")
	require.NoError(t, err)

	zcore, logObserver := observer.New(zapcore.InfoLevel)
	log := zap.New(zcore)
	onChange := func() {
		log.Info("Content changed")
	}

	err = w.WatchFiles([]string{testFile.Name()}, onChange, log)
	require.NoError(t, err)

	testFile.WriteString("test content changed")
	assertLogs(t,
		func() bool {
			return logObserver.FilterMessage("Content changed").Len() > 0
		},
		"Unable to locate 'Content changed' in log. All logs: %v", logObserver)

	os.Remove(testFile.Name())
	assertLogs(t,
		func() bool {
			return logObserver.FilterMessage(testFile.Name()+"has been removed.").Len() > 0
		},
		"Unable to locate 'Content changed' in log. All logs: %v", logObserver)

	err = w.Close()
	assert.NoError(t, err)
}

type delayedFormat struct {
	fn func() interface{}
}

func (df delayedFormat) String() string {
	return fmt.Sprintf("%v", df.fn())
}

func assertLogs(t *testing.T, f func() bool, errorMsg string, logObserver *observer.ObservedLogs) {
	assert.Eventuallyf(t, f,
		10*time.Second, 10*time.Millisecond,
		errorMsg,
		delayedFormat{
			fn: func() interface{} { return logObserver.All() },
		},
	)
}

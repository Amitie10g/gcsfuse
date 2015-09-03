// Copyright 2015 Google Inc. All Rights Reserved.
//
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

package integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/wiring"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestGcsfuse(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type GcsfuseTest struct {
	// Path to the gcsfuse binary.
	gcsfusePath string

	// A temporary directory into which a file system may be mounted. Removed in
	// TearDown.
	dir string
}

var _ SetUpInterface = &GcsfuseTest{}
var _ TearDownInterface = &GcsfuseTest{}

func init() { RegisterTestSuite(&GcsfuseTest{}) }

func (t *GcsfuseTest) SetUp(_ *TestInfo) {
	var err error
	t.gcsfusePath = path.Join(gBuildDir, "bin/gcsfuse")

	// Set up the temporary directory.
	t.dir, err = ioutil.TempDir("", "gcsfuse_test")
	AssertEq(nil, err)
}

func (t *GcsfuseTest) TearDown() {
	err := os.Remove(t.dir)
	AssertEq(nil, err)
}

// Call gcsfuse with the supplied args, waiting for it to mount. Return nil
// only if it mounts successfully.
func (t *GcsfuseTest) mount(args []string) (err error) {
	// Set up a pipe that gcsfuse can write to to tell us when it has
	// successfully mounted.
	statusR, statusW, err := os.Pipe()
	if err != nil {
		err = fmt.Errorf("Pipe: %v", err)
		return
	}

	// Run gcsfuse, writing the result of waiting for it to a channel.
	gcsfuseErr := make(chan error, 1)
	go func() {
		gcsfuseErr <- t.runGcsfuse(args, statusW)
	}()

	// In the background, wait for something to be written to the pipe.
	pipeErr := make(chan error, 1)
	go func() {
		defer statusR.Close()
		n, err := statusR.Read(make([]byte, 1))
		if n == 1 {
			pipeErr <- nil
			return
		}

		pipeErr <- fmt.Errorf("statusR.Read: %v", err)
	}()

	// Watch for a result from one of them.
	select {
	case err = <-gcsfuseErr:
		err = fmt.Errorf("gcsfuse: %v", err)
		return

	case err = <-pipeErr:
		if err == nil {
			// All is good.
			return
		}

		err = <-gcsfuseErr
		err = fmt.Errorf("gcsfuse after pipe error: %v", err)
		return
	}
}

// Run gcsfuse and wait for it to return. Hand it the supplied pipe to write
// into when it successfully mounts. This function takes responsibility for
// closing the write end of the pipe locally.
func (t *GcsfuseTest) runGcsfuse(args []string, statusW *os.File) (err error) {
	defer statusW.Close()

	cmd := exec.Command(t.gcsfusePath)
	cmd.Args = append(cmd.Args, args...)
	cmd.ExtraFiles = []*os.File{statusW}
	cmd.Env = []string{"STATUS_PIPE=3"}

	output, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("%v\nOutput:\n%s", err, output)
		return
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *GcsfuseTest) BadUsage() {
	testCases := []struct {
		args           []string
		expectedOutput string
	}{
		// Too few args
		0: {
			[]string{wiring.FakeBucket},
			"exactly two arguments",
		},

		// Too many args
		1: {
			[]string{wiring.FakeBucket, "a", "b"},
			"exactly two arguments",
		},

		// Unknown flag
		2: {
			[]string{"--tweak_frobnicator", wiring.FakeBucket, "a"},
			"not defined.*tweak_frobnicator",
		},
	}

	// Run each test case.
	for i, tc := range testCases {
		cmd := exec.Command(t.gcsfusePath)
		cmd.Args = append(cmd.Args, tc.args...)

		output, err := cmd.CombinedOutput()
		ExpectThat(err, Error(HasSubstr("exit status")), "case %d", i)
		ExpectThat(string(output), MatchesRegexp(tc.expectedOutput), "case %d", i)
	}
}

func (t *GcsfuseTest) ReadOnlyMode() {
	var err error

	// Mount.
	args := []string{"-o", "ro", wiring.FakeBucket, t.dir}

	err = t.mount(args)
	AssertEq(nil, err)

	// Check that the expected file is there (cf. the documentation on
	// wiring.FakeBucket).
	contents, err := ioutil.ReadFile(path.Join(t.dir, "foo"))
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))

	// The implicit directory shouldn't be visible, since we don't have implicit
	// directories enabled.
	_, err = os.Lstat(path.Join(t.dir, "bar"))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)

	// Writing to the file system should ail.
	err = ioutil.WriteFile(path.Join(t.dir, "blah"), []byte{}, 0400)
	ExpectThat(err, Error(HasSubstr("TODO")))
}

func (t *GcsfuseTest) ReadWriteMode() {
	AssertTrue(false, "TODO")
}

func (t *GcsfuseTest) FileAndDirModeFlags() {
	AssertTrue(false, "TODO")
}

func (t *GcsfuseTest) UidAndGidFlags() {
	AssertTrue(false, "TODO")
}

func (t *GcsfuseTest) ImplicitDirs() {
	AssertTrue(false, "TODO")
}

func (t *GcsfuseTest) VersionFlags() {
	AssertTrue(false, "TODO")
}

func (t *GcsfuseTest) HelpFlags() {
	AssertTrue(false, "TODO")
}

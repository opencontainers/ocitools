package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	tap "github.com/mndrix/tap-go"
	"github.com/mrunalp/fileutils"
	rspec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/specerror"
	"github.com/opencontainers/runtime-tools/validation/util"
	uuid "github.com/satori/go.uuid"
)

func main() {
	t := tap.New()
	t.Header(0)

	// Create a cgroup
	cgPath := "/sys/fs/cgroup"
	testPath := filepath.Join(cgPath, "pids", "cgrouptest")
	os.Mkdir(testPath, 0755)
	defer os.RemoveAll(testPath)

	bundleDir, err := util.PrepareBundle()
	if err != nil {
		util.Fatal(err)
	}
	defer os.RemoveAll(bundleDir)

	r, err := util.NewRuntime(util.RuntimeCommand, bundleDir)
	if err != nil {
		util.Fatal(err)
	}

	uid, err := uuid.NewV4()
	if err != nil {
		util.Fatal(err)
	}
	r.SetID(uid.String())
	g, err := util.GetDefaultGenerator()
	if err != nil {
		util.Fatal(err)
	}

	err = r.SetConfig(g)
	if err != nil {
		util.Fatal(err)
	}
	err = fileutils.CopyFile("runtimetest", filepath.Join(r.BundleDir, "runtimetest"))
	if err != nil {
		util.Fatal(err)
	}

	err = r.Create()
	if err != nil {
		util.Fatal(err)
	}

	state, err := r.State()
	if err != nil {
		util.Fatal(err)
	}
	// Add the container to the cgroup
	err = ioutil.WriteFile(filepath.Join(testPath, "tasks"), []byte(strconv.Itoa(state.Pid)), 0644)
	if err != nil {
		util.Fatal(err)
	}

	err = r.Start()
	if err != nil {
		util.Fatal(err)
	}

	err = util.WaitingForStatus(r, util.LifecycleStatusStopped, time.Second*10, time.Second*1)
	if err == nil {
		err = r.Delete()
	}
	if err != nil {
		t.Fail(err.Error())
	}

	_, err = os.Stat(testPath)
	fmt.Println(err)
	util.SpecErrorOK(t, err == nil, specerror.NewError(specerror.DeleteOnlyCreatedRes, fmt.Errorf("Note that resources associated with the container, but not created by this container, MUST NOT be deleted"), rspec.Version), nil)

	t.AutoPlan()
}

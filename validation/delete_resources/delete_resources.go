package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tap "github.com/mndrix/tap-go"
	"github.com/mrunalp/fileutils"
	rspec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/cgroups"
	"github.com/opencontainers/runtime-tools/specerror"
	"github.com/opencontainers/runtime-tools/validation/util"
	uuid "github.com/satori/go.uuid"
)

func main() {
	t := tap.New()
	t.Header(0)

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

	var limit int64 = 1000

	g.SetLinuxCgroupsPath(cgroups.AbsCgroupPath)
	g.SetLinuxResourcesPidsLimit(limit)

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
	if err := util.ValidateLinuxResourcesPids(g.Spec(), t, &state); err != nil {
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

	path := filepath.Join("/sys/fs/cgroup/pids", cgroups.AbsCgroupPath)
	_, err = os.Stat(path)
	util.SpecErrorOK(t, os.IsNotExist(err), specerror.NewError(specerror.DeleteResImplement, fmt.Errorf("Deleting a container MUST delete the resources that were created during the `create` step"), rspec.Version), nil)

	t.AutoPlan()
}

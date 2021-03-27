package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	tap "github.com/mndrix/tap-go"
	"github.com/opencontainers/runtime-tools/validation/util"
	uuid "github.com/satori/go.uuid"
)

func main() {
	t := tap.New()
	t.Header(0)

	tempDir, err := ioutil.TempDir("", "oci-pid")
	if err != nil {
		util.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	tempPidFile := filepath.Join(tempDir, "pidfile")

	g, err := util.GetDefaultGenerator()
	if err != nil {
		util.Fatal(err)
	}
	g.SetProcessArgs([]string{"true"})
	config := util.LifecycleConfig{
		Config:  g,
		Actions: util.LifecycleActionCreate | util.LifecycleActionStart | util.LifecycleActionDelete,
		PreCreate: func(r *util.Runtime) error {
			uid, err := uuid.NewV4()
			if err != nil {
				util.Fatal(err)
			}
			r.SetID(uid.String())
			r.PidFile = tempPidFile
			return nil
		},
		PostCreate: func(r *util.Runtime) error {
			pidData, err := ioutil.ReadFile(tempPidFile)
			if err != nil {
				return err
			}
			pid, err := strconv.Atoi(string(pidData))
			if err != nil {
				return err
			}
			state, err := r.State()
			if err != nil {
				return err
			}
			if state.Pid != pid {
				return fmt.Errorf("wrong pid %d, expected %d", pid, state.Pid)
			}
			return nil
		},
		PreDelete: func(r *util.Runtime) error {
			util.WaitingForStatus(*r, util.LifecycleStatusRunning, time.Second*10, time.Second*1)
			err = r.Kill("KILL")
			// wait before the container been deleted
			util.WaitingForStatus(*r, util.LifecycleStatusStopped, time.Second*10, time.Second*1)
			return err
		},
	}

	err = util.RuntimeLifecycleValidate(config)
	t.Ok(err == nil, "create with '--pid-file' option works")
	if err != nil {
		diagnostic := map[string]string{
			"error": err.Error(),
		}
		if e, ok := err.(*exec.ExitError); ok {
			if len(e.Stderr) > 0 {
				diagnostic["stderr"] = string(e.Stderr)
			}
		}
		t.YAML(diagnostic)
	}

	t.AutoPlan()
}

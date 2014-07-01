package runner

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/cloudfoundry-incubator/garden/client"
	"github.com/cloudfoundry-incubator/garden/client/connection"
	"github.com/cloudfoundry-incubator/garden/warden"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega/gexec"
)

type Runner struct {
	addr string

	bin  string
	argv []string

	binPath    string
	rootFSPath string

	tmpdir string
}

func New(addr string, bin, binPath, rootFSPath string, argv ...string) *Runner {
	return &Runner{
		addr: addr,

		bin:  bin,
		argv: argv,

		binPath:    binPath,
		rootFSPath: rootFSPath,

		tmpdir: filepath.Join(
			os.TempDir(),
			fmt.Sprintf("test-warden-%d", ginkgo.GinkgoParallelNode()),
		),
	}
}

func (r *Runner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	err := os.MkdirAll(r.tmpdir, 0755)
	if err != nil {
		return err
	}

	depotPath := filepath.Join(r.tmpdir, "containers")
	overlaysPath := filepath.Join(r.tmpdir, "overlays")
	snapshotsPath := filepath.Join(r.tmpdir, "snapshots")
	graphPath := filepath.Join(r.tmpdir, "graph")

	if err := os.MkdirAll(depotPath, 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(snapshotsPath, 0755); err != nil {
		return err
	}

	wardenArgs := append(
		r.argv,
		"--listenNetwork", "tcp",
		"--listenAddr", r.addr,
		"--bin", r.binPath,
		"--rootfs", r.rootFSPath,
		"--depot", depotPath,
		"--overlays", overlaysPath,
		"--snapshots", snapshotsPath,
		"--graph", graphPath,
		"--debug",
		"--disableQuotas",
		"--networkPool", fmt.Sprintf("10.250.%d.0/24", ginkgo.GinkgoParallelNode()),
		"--portPoolStart", strconv.Itoa(51000+(1000*ginkgo.GinkgoParallelNode())),
		"--portPoolSize", "1000",
		"--uidPoolStart", strconv.Itoa(10000*ginkgo.GinkgoParallelNode()),
		"--tag", strconv.Itoa(ginkgo.GinkgoParallelNode()),
	)

	session, err := gexec.Start(
		exec.Command(r.bin, wardenArgs...),
		gexec.NewPrefixedWriter("\x1b[32m[o]\x1b[31m[warden-linux]\x1b[0m ", ginkgo.GinkgoWriter),
		gexec.NewPrefixedWriter("\x1b[91m[e]\x1b[31m[warden-linux]\x1b[0m ", ginkgo.GinkgoWriter),
	)
	if err != nil {
		return err
	}

	var dialErr error
	for i := 0; i < 10; i++ {
		dialErr = r.TryDial()
		if dialErr == nil {
			break
		}

		time.Sleep(time.Second)
	}

	if dialErr != nil {
		return err
	}

	close(ready)

	var signal os.Signal

dance:
	for {
		select {
		case signal = <-signals:
			if signal == syscall.SIGKILL {
				if err := r.destroyContainers(); err != nil {
					return err
				}
			}

			session.Signal(syscall.SIGTERM)
		case <-session.Exited:
			break dance
		}
	}

	if signal == syscall.SIGKILL {
		if err := os.RemoveAll(r.tmpdir); err != nil {
			return err
		}
	}

	if session.ExitCode() == 0 {
		return nil
	}

	return fmt.Errorf("exit status %d", session.ExitCode())
}

func (r *Runner) TryDial() error {
	conn, dialErr := net.Dial("tcp", r.addr)

	if dialErr == nil {
		conn.Close()
		return nil
	}

	return dialErr
}

func (r *Runner) NewClient() warden.Client {
	return client.New(connection.New("tcp", r.addr))
}

func (r *Runner) destroyContainers() error {
	client := r.NewClient()

	containers, err := client.Containers(nil)
	if err != nil {
		return err
	}

	for _, container := range containers {
		err := client.Destroy(container.Handle())
		if err != nil {
			return err
		}
	}

	return nil
}

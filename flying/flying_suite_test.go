package flying_test

import (
	"net/http"
	"os"

	"github.com/concourse/testflight/bosh"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
	"time"
)

var flyBin string

var _ = BeforeSuite(func() {
	var err error

	flyBin, err = gexec.Build("github.com/concourse/fly", "-race")
	Ω(err).ShouldNot(HaveOccurred())

	bosh.DeleteDeployment("concourse")

	bosh.Deploy("noop.yml")

	atcURL := "http://10.244.8.2:8080"

	os.Setenv("ATC_URL", atcURL)

	Eventually(errorPolling(atcURL), 1*time.Minute).ShouldNot(HaveOccurred())
})

func TestFlying(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Flying Suite")
}

func errorPolling(url string) func() error {
	return func() error {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
		}

		return err
	}
}

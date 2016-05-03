package pipelines_test

import (
	"github.com/concourse/testflight/gitserver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Configuring a resource in a pipeline config", func() {
	var gitServer *gitserver.Server

	BeforeEach(func() {
		gitServer = gitserver.Start(client)
		gitServer.CommitResource()

		configurePipeline(
			"-c", "fixtures/config_params.yml",
			"-v", "git-server="+gitServer.URI(),
		)
	})

	AfterEach(func() {
		gitServer.Stop()
	})

	Context("when specifying file in task config", func() {
		BeforeEach(func() {
			taskFileContents := `---
platform: linux
run:
  path: sh
  args: ["-c", "printenv SOURCE_PARAM"]
params:
  SOURCE_PARAM: file_source
`
			gitServer.CommitFileToBranch(taskFileContents, "task.yml", "master")
		})

		It("executes the file with params specified in file", func() {
			watch := flyWatch("file-test")
			Expect(watch).To(gbytes.Say("file_source"))
		})

		It("executes the file with params from config", func() {
			watch := flyWatch("file-config-params-test")
			Expect(watch).To(gbytes.Say("config_params_source"))
		})

		It("executes the file with job params", func() {
			watch := flyWatch("file-params-test")
			Expect(watch).To(gbytes.Say("job_params_source"))
		})

		It("executes the file with job params, overlaying the config params", func() {
			watch := flyWatch("everything-params-test")
			Expect(watch).To(gbytes.Say("job_params_source"))
		})
	})
})

package pipelines_test

import (
	"github.com/concourse/testflight/gitserver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Configuring a resource in a pipeline config", func() {
	var originGitServer *gitserver.Server

	BeforeEach(func() {
		originGitServer = gitserver.Start(client)
		originGitServer.CommitResource()
		originGitServer.CommitFileToBranch("initial", "initial", "trigger")

		configurePipeline(
			"-c", "fixtures/resource-types.yml",
			"-v", "origin-git-server="+originGitServer.URI(),
		)
	})

	AfterEach(func() {
		originGitServer.Stop()
	})

	It("can use custom resource types for 'get', 'put', and task 'image_resource's", func() {
		watch := flyWatch("resource-getter")
		Expect(watch).To(gbytes.Say("rootfs/some-file"))

		watch = flyWatch("resource-putter")
		Expect(watch).To(gbytes.Say("pushing using custom resource"))
		Expect(watch).To(gbytes.Say("some-output/some-file"))

		watch = flyWatch("resource-imgur")
		Expect(watch).To(gbytes.Say("fetched from custom resource"))
		Expect(watch).To(gbytes.Say("SOME_ENV=yep"))
	})
})

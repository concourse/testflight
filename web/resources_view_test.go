package web_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	// . "github.com/sclevine/agouti/matchers"

	"github.com/concourse/atc"
)

var _ = Describe("Viewing resources", func() {
	Describe("a broken resource", func() {
		var brokenResource atc.Resource

		BeforeEach(func() {
			_, _, _, err := client.CreateOrUpdatePipelineConfig(pipelineName, "0", atc.Config{
				Resources: []atc.ResourceConfig{
					{
						Name: "broken-resource",
						Type: "git",
						Source: atc.Source{
							"branch": "master",
							"uri":    "i r not reall?",
						},
						CheckEvery: "",
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = client.UnpausePipeline(pipelineName)
			Expect(err).NotTo(HaveOccurred())

			var found bool
			brokenResource, found, err = client.Resource(pipelineName, "broken-resource")
			Expect(found).To(BeTrue())
			Expect(err).NotTo(HaveOccurred())
		})

		It("correctly displays logs", func() {
			url := atcRoute(fmt.Sprintf("/teams/%s/pipelines/%s/resources/%s", teamName, pipelineName, brokenResource.Name))

			counter := 0
			for {
				Expect(page.Navigate(url)).To(Succeed())
				if counter == 120 {
					Fail("Unable to locate resource log information.")
				}

				if visible, _ := page.Find(".resource-check-status .header i.errored").Visible(); visible {
					break
				}
				counter++
				time.Sleep(500 * time.Millisecond)
			}

			Eventually(page.Find("pre").Text).Should(ContainSubstring("failed: exit status"))
		})
	})
})

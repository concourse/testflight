package web_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/sclevine/agouti/matchers"

	"github.com/concourse/atc"
)

var _ = Describe("Aborting a build", func() {
	Context("with a build in the configuration", func() {
		var build atc.Build

		BeforeEach(func() {
			_, _, err := client.CreateOrUpdatePipelineConfig(pipelineName, "0", atc.Config{
				Jobs: []atc.JobConfig{
					{
						Name: "some-job",
						Plan: atc.PlanSequence{
							{
								Task: "some-task",
								TaskConfig: &atc.TaskConfig{
									Run: atc.TaskRunConfig{
										Path: "sleep",
										Args: []string{"1000"},
									},
								},
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = client.UnpausePipeline(pipelineName)
			Expect(err).NotTo(HaveOccurred())

			build, err = client.CreateJobBuild(pipelineName, "some-job")
			Expect(err).NotTo(HaveOccurred())
		})

		It("can abort the build", func() {
			Expect(page.Navigate(atcRoute(build.URL))).To(Succeed())
			Eventually(page).Should(HaveURL(atcRoute(fmt.Sprintf("pipelines/%s/jobs/some-job/builds/%s", pipelineName, build.Name))))
			Eventually(page.Find("h1")).Should(HaveText(fmt.Sprintf("some-job #%s", build.Name)))

			Eventually(page.Find(".build-action-abort")).Should(BeFound())
			Expect(page.Find(".build-action-abort").Click()).To(Succeed())

			Eventually(page.Find("#page-header.aborted")).Should(BeFound())
			Eventually(page.Find(".build-action-abort")).ShouldNot(BeFound())
		})
	})

	Context("with a one-off build", func() {
		var build atc.Build

		BeforeEach(func() {
			var err error

			pf := atc.NewPlanFactory(0)

			build, err = client.CreateBuild(pf.NewPlan(atc.TaskPlan{
				Name: "some-task",
				Config: &atc.TaskConfig{
					Run: atc.TaskRunConfig{
						Path: "sleep",
						Args: []string{"1000"},
					},
				},
			}))
			Expect(err).NotTo(HaveOccurred())
		})

		It("can abort the build", func() {
			Expect(page.Navigate(atcRoute(build.URL))).To(Succeed())
			Eventually(page).Should(HaveURL(atcRoute(fmt.Sprintf("builds/%d", build.ID))))
			Eventually(page.Find("h1")).Should(HaveText(fmt.Sprintf("build #%d", build.ID)))

			Eventually(page.Find(".build-action-abort")).Should(BeFound())
			Expect(page.Find(".build-action-abort").Click()).To(Succeed())

			Eventually(page.Find("#page-header.aborted")).Should(BeFound())
			Eventually(page.Find(".build-action-abort")).ShouldNot(BeFound())
		})
	})
})

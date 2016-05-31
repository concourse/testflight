package web_test

import (
	"fmt"
	"time"

	"github.com/concourse/atc"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/sclevine/agouti/matchers"
)

var _ = Describe("PipelinePausing", func() {
	var (
		loadingTimeout time.Duration
	)

	Context("with a job in the configuration", func() {
		BeforeEach(func() {
			_, _, _, err := team.CreateOrUpdatePipelineConfig(pipelineName, "0", atc.Config{
				Jobs: []atc.JobConfig{
					{Name: "some-job-name"},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			_, err = team.UnpausePipeline(pipelineName)
			Expect(err).NotTo(HaveOccurred())

			_, err = team.DeletePipeline("another-pipeline")
			Expect(err).NotTo(HaveOccurred())

			_, _, _, err = team.CreateOrUpdatePipelineConfig("another-pipeline", "0", atc.Config{
				Jobs: []atc.JobConfig{
					{Name: "another-job-name"},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = team.UnpausePipeline("another-pipeline")
			Expect(err).NotTo(HaveOccurred())

			loadingTimeout = 10 * time.Second
		})

		AfterEach(func() {
			_, err := team.DeletePipeline("another-pipeline")
			Expect(err).NotTo(HaveOccurred())
		})

		homeLink := ".js-groups li:nth-of-type(2) a"
		navList := ".js-pipelinesNav-list"

		It("can pause the pipelines", func() {
			Expect(page.Navigate(atcURL)).To(Succeed())
			Eventually(page, loadingTimeout).Should(HaveURL(atcRoute("/")))

			By("toggling the nav")
			Expect(page.Find(".js-pipelinesNav-toggle").Click()).To(Succeed())

			By("clicking another-pipeline")
			Eventually(page.All(navList).FindByLink("another-pipeline")).Should(BeFound())
			Expect(page.All(navList).FindByLink("another-pipeline").Click()).To(Succeed())
			Eventually(page, loadingTimeout).Should(HaveURL(atcRoute(fmt.Sprintf("/teams/%s/pipelines/another-pipeline", teamName))))

			By("clicking home button")
			Expect(page.Find(homeLink).Click()).To(Succeed())
			Eventually(page, loadingTimeout).Should(HaveURL(atcRoute(fmt.Sprintf("/teams/%s/pipelines/another-pipeline", teamName))))

			By("toggling the nav")
			Expect(page.Find(".js-pipelinesNav-toggle").Click()).To(Succeed())
			Eventually(page.Find("#pipeline").Text, loadingTimeout).Should(ContainSubstring("another-job-name"))

			By("pausing another-pipeline")
			spanXPath := fmt.Sprintf("//a[@href='/teams/%s/pipelines/another-pipeline']/parent::li/span", teamName)
			Eventually(page.All(navList).FindByXPath(spanXPath), loadingTimeout).Should(BeVisible())
			Expect(page.All(navList).FindByXPath(spanXPath + "[contains(@class, 'disabled')]")).To(BeFound())
			Expect(page.FindByXPath(spanXPath).Click()).To(Succeed())

			// top bar should show the pipeline is paused
			Eventually(page.Find(".js-groups.paused"), loadingTimeout).Should(BeFound())

			By("refreshing the page")
			page.Refresh()

			Eventually(page.Find(".js-groups.paused"), loadingTimeout).Should(BeFound())
			Expect(page.Find(".js-pipelinesNav-toggle").Click()).To(Succeed())

			Eventually(page.All(navList).FindByXPath(spanXPath), loadingTimeout).Should(BeVisible())
			Expect(page.All(navList).FindByXPath(spanXPath + "[contains(@class, 'enabled')]")).To(BeFound())

			By("unpausing the pipeline")
			Expect(page.FindByXPath(spanXPath).Click()).To(Succeed())
			Eventually(page.All(navList).FindByXPath(spanXPath + "[contains(@class, 'disabled')]")).Should(BeFound())

			Consistently(page.Find(".js-groups.paused")).ShouldNot(BeFound())

			By("refreshing the page")
			page.Refresh()

			By("pausing the pipeline")
			Expect(page.Find(".js-pipelinesNav-toggle").Click()).To(Succeed())
			Expect(page.FindByXPath(spanXPath).Click()).To(Succeed())
			Eventually(page.All(navList).FindByXPath(spanXPath + "[contains(@class, 'enabled')]")).Should(BeFound())
		})
	})
})

package flying_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"syscall"

	"github.com/concourse/testflight/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Flying", func() {
	var tmpdir string
	var fixture, input1, input2 string

	BeforeEach(func() {
		var err error

		tmpdir, err = ioutil.TempDir("", "fly-test")
		Expect(err).NotTo(HaveOccurred())

		fixture = filepath.Join(tmpdir, "fixture")
		input1 = filepath.Join(tmpdir, "input-1")
		input2 = filepath.Join(tmpdir, "input-2")

		err = os.MkdirAll(fixture, 0755)
		Expect(err).NotTo(HaveOccurred())

		err = os.MkdirAll(input1, 0755)
		Expect(err).NotTo(HaveOccurred())

		err = os.MkdirAll(input2, 0755)
		Expect(err).NotTo(HaveOccurred())

		err = ioutil.WriteFile(
			filepath.Join(fixture, "run"),
			[]byte(`#!/bin/sh
echo some output
echo FOO is $FOO
echo ARGS are "$@"
exit 0
`),
			0755,
		)
		Expect(err).NotTo(HaveOccurred())

		err = ioutil.WriteFile(
			filepath.Join(tmpdir, "task.yml"),
			[]byte(`---
platform: linux

image_resource:
  type: docker-image
  source: {repository: busybox}

inputs:
- name: fixture
- name: input-1
- name: input-2

outputs:
- name: output-1
- name: output-2

params:
  FOO: 1

run:
  path: fixture/run
`),
			0644,
		)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpdir)
	})

	It("works", func() {
		fly := exec.Command(flyBin, "-t", targetedConcourse, "execute", "-c", "task.yml", "-i", "fixture=./fixture", "-i", "input-1=./input-1", "-i", "input-2=./input-2", "--", "SOME", "ARGS")
		fly.Dir = tmpdir

		session := helpers.StartFly(fly)

		Eventually(session).Should(gexec.Exit(0))

		Expect(session).To(gbytes.Say("some output"))
		Expect(session).To(gbytes.Say("FOO is 1"))
		Expect(session).To(gbytes.Say("ARGS are SOME ARGS"))
	})

	Describe("hijacking", func() {
		It("executes an interactive command in a running task's container", func() {
			err := ioutil.WriteFile(
				filepath.Join(fixture, "run"),
				[]byte(`#!/bin/sh
mkfifo /tmp/fifo
echo waiting
cat < /tmp/fifo
`),
				0755,
			)
			Expect(err).NotTo(HaveOccurred())

			fly := exec.Command(flyBin, "-t", targetedConcourse, "execute", "-c", "task.yml", "-i", "fixture=./fixture", "-i", "input-1=./input-1", "-i", "input-2=./input-2")
			fly.Dir = tmpdir

			flyS := helpers.StartFly(fly)

			Eventually(flyS).Should(gbytes.Say("executing build"))

			buildRegex := regexp.MustCompile(`executing build (\d+)`)
			matches := buildRegex.FindSubmatch(flyS.Out.Contents())
			buildID := string(matches[1])

			Eventually(flyS).Should(gbytes.Say("waiting"))

			env := exec.Command(flyBin, "-t", targetedConcourse, "hijack", "-b", buildID, "-s", "one-off", "--", "env")
			envS := helpers.StartFly(env)
			<-envS.Exited
			Expect(envS.ExitCode()).To(Equal(0))
			Expect(envS.Out).To(gbytes.Say("FOO=1"))

			hijack := exec.Command(flyBin, "-t", targetedConcourse, "hijack", "-b", buildID, "-s", "one-off", "--", "sh", "-c", "echo marco > /tmp/fifo")
			hijackS := helpers.StartFly(hijack)
			Eventually(flyS).Should(gbytes.Say("marco"))
			Eventually(hijackS).Should(gexec.Exit())
			Eventually(flyS).Should(gexec.Exit(0))
		})
	})

	Describe("uploading inputs with and without -x", func() {
		BeforeEach(func() {
			gitIgnorePath := filepath.Join(input1, ".gitignore")

			err := ioutil.WriteFile(gitIgnorePath, []byte(`*.exist`), 0644)
			Expect(err).NotTo(HaveOccurred())

			fileToBeIgnoredPath := filepath.Join(input1, "expect-not-to.exist")
			err = ioutil.WriteFile(fileToBeIgnoredPath, []byte(`ignored file content`), 0644)
			Expect(err).NotTo(HaveOccurred())

			fileToBeIncludedPath := filepath.Join(input2, "expect-to.exist")
			err = ioutil.WriteFile(fileToBeIncludedPath, []byte(`included file content`), 0644)
			Expect(err).NotTo(HaveOccurred())

			file1 := filepath.Join(input1, "file-1")
			err = ioutil.WriteFile(file1, []byte(`file-1 contents`), 0644)
			Expect(err).NotTo(HaveOccurred())

			file2 := filepath.Join(input2, "file-2")
			err = ioutil.WriteFile(file2, []byte(`file-2 contents`), 0644)
			Expect(err).NotTo(HaveOccurred())

			err = os.Mkdir(filepath.Join(input1, ".git"), 0755)
			Expect(err).NotTo(HaveOccurred())

			err = os.Mkdir(filepath.Join(input1, ".git/refs"), 0755)
			Expect(err).NotTo(HaveOccurred())

			err = os.Mkdir(filepath.Join(input1, ".git/objects"), 0755)
			Expect(err).NotTo(HaveOccurred())

			gitHEADPath := filepath.Join(input1, ".git/HEAD")
			err = ioutil.WriteFile(gitHEADPath, []byte(`ref: refs/heads/master`), 0644)
			Expect(err).NotTo(HaveOccurred())

			err = ioutil.WriteFile(
				filepath.Join(fixture, "run"),
				[]byte(`#!/bin/sh
cp -a input-1/. output-1/
cp -a input-2/. output-2/
`),
				0755,
			)
			Expect(err).NotTo(HaveOccurred())
		})

		It("uploads git repo input and non git repo input, IGNORING things in the .gitignore for git repo inputs", func() {
			fly := exec.Command(flyBin, "-t", targetedConcourse, "execute", "-c", "task.yml", "-i", "fixture=./fixture", "-i", "input-1=./input-1", "-i", "input-2=./input-2", "-o", "output-1=./output-1", "-o", "output-2=./output-2")
			fly.Dir = tmpdir

			session := helpers.StartFly(fly)
			<-session.Exited

			Expect(session.ExitCode()).To(Equal(0))

			fileToBeIgnoredPath := filepath.Join(tmpdir, "output-1", "expect-not-to.exist")
			fileToBeIncludedPath := filepath.Join(tmpdir, "output-2", "expect-to.exist")
			file1 := filepath.Join(tmpdir, "output-1", "file-1")
			file2 := filepath.Join(tmpdir, "output-2", "file-2")

			_, err := ioutil.ReadFile(fileToBeIgnoredPath)
			Expect(err).To(HaveOccurred())

			Expect(ioutil.ReadFile(fileToBeIncludedPath)).To(Equal([]byte("included file content")))
			Expect(ioutil.ReadFile(file1)).To(Equal([]byte("file-1 contents")))
			Expect(ioutil.ReadFile(file2)).To(Equal([]byte("file-2 contents")))
		})

		It("uploads git repo input and non git repo input, INCLUDING things in the .gitignore for git repo inputs", func() {
			fly := exec.Command(flyBin, "-t", targetedConcourse, "execute", "-x", "-c", "task.yml", "-i", "fixture=./fixture", "-i", "input-1=./input-1", "-i", "input-2=./input-2", "-o", "output-1=./output-1", "-o", "output-2=./output-2")
			fly.Dir = tmpdir

			session := helpers.StartFly(fly)
			<-session.Exited

			Expect(session.ExitCode()).To(Equal(0))

			fileToBeIgnoredPath := filepath.Join(tmpdir, "output-1", "expect-not-to.exist")
			fileToBeIncludedPath := filepath.Join(tmpdir, "output-2", "expect-to.exist")
			file1 := filepath.Join(tmpdir, "output-1", "file-1")
			file2 := filepath.Join(tmpdir, "output-2", "file-2")

			Expect(ioutil.ReadFile(fileToBeIgnoredPath)).To(Equal([]byte("ignored file content")))
			Expect(ioutil.ReadFile(fileToBeIncludedPath)).To(Equal([]byte("included file content")))
			Expect(ioutil.ReadFile(file1)).To(Equal([]byte("file-1 contents")))
			Expect(ioutil.ReadFile(file2)).To(Equal([]byte("file-2 contents")))
		})
	})

	Describe("pulling down outputs", func() {
		It("works", func() {
			err := ioutil.WriteFile(
				filepath.Join(fixture, "run"),
				[]byte(`#!/bin/sh
echo hello > output-1/file-1
echo world > output-2/file-2
`),
				0755,
			)
			Expect(err).NotTo(HaveOccurred())

			fly := exec.Command(flyBin, "-t", targetedConcourse, "execute", "-c", "task.yml", "-i", "fixture=./fixture", "-i", "input-1=./input-1", "-i", "input-2=./input-2", "-o", "output-1=./output-1", "-o", "output-2=./output-2")
			fly.Dir = tmpdir

			session := helpers.StartFly(fly)
			<-session.Exited

			Expect(session.ExitCode()).To(Equal(0))

			file1 := filepath.Join(tmpdir, "output-1", "file-1")
			file2 := filepath.Join(tmpdir, "output-2", "file-2")

			Expect(ioutil.ReadFile(file1)).To(Equal([]byte("hello\n")))
			Expect(ioutil.ReadFile(file2)).To(Equal([]byte("world\n")))
		})
	})

	Describe("aborting", func() {
		It("terminates the running task", func() {
			err := ioutil.WriteFile(
				filepath.Join(fixture, "run"),
				[]byte(`#!/bin/sh
trap "echo task got sigterm; exit 1" SIGTERM
sleep 1000 &
echo waiting
wait
`),
				0755,
			)
			Expect(err).NotTo(HaveOccurred())

			fly := exec.Command(flyBin, "-t", targetedConcourse, "execute", "-c", "task.yml", "-i", "fixture=./fixture", "-i", "input-1=./input-1", "-i", "input-2=./input-2")
			fly.Dir = tmpdir

			flyS := helpers.StartFly(fly)

			Eventually(flyS).Should(gbytes.Say("waiting"))

			flyS.Signal(syscall.SIGTERM)

			Eventually(flyS).Should(gbytes.Say("task got sigterm"))

			// build should have been aborted
			Eventually(flyS).Should(gexec.Exit(3))
		})
	})
})

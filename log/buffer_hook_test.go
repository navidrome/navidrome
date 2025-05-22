package log

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("bufferHook", func() {
	var hook *bufferHook
	var originalBuffer *RingBuffer

	BeforeEach(func() {
		hook = &bufferHook{}
		// Save original log buffer
		originalBuffer = logBuffer
		// Reset buffer for testing
		logBuffer = NewRingBuffer(5)
	})

	AfterEach(func() {
		// Restore original log buffer
		logBuffer = originalBuffer
	})

	It("should implement the logrus.Hook interface", func() {
		var _ logrus.Hook = hook
	})

	It("should register for all log levels", func() {
		levels := hook.Levels()
		Expect(levels).To(Equal(logrus.AllLevels))
	})

	Context("when firing an entry", func() {
		BeforeEach(func() {
			entry := &logrus.Entry{
				Message: "test message",
				Level:   logrus.InfoLevel,
			}
			err := hook.Fire(entry)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should add the entry to the ring buffer", func() {
			entries := logBuffer.GetAll()
			Expect(entries).To(HaveLen(1))
			Expect(entries[0].Message).To(Equal("test message"))
			Expect(entries[0].Level).To(Equal(logrus.InfoLevel))
		})
	})

	Context("when using listener functions", func() {
		var ch chan *logrus.Entry

		BeforeEach(func() {
			ch = make(chan *logrus.Entry, 1)
			RegisterLogListener(ch)
		})

		AfterEach(func() {
			UnregisterLogListener(ch)
		})

		It("should broadcast log entries to registered listeners", func() {
			entry := &logrus.Entry{
				Message: "broadcast test",
				Level:   logrus.InfoLevel,
			}
			err := hook.Fire(entry)
			Expect(err).NotTo(HaveOccurred())

			// Check if the entry was received by the listener
			var received *logrus.Entry
			Eventually(ch).Should(Receive(&received))
			Expect(received.Message).To(Equal("broadcast test"))
		})

		It("should successfully unregister listeners", func() {
			UnregisterLogListener(ch)

			entry := &logrus.Entry{
				Message: "after unregister",
				Level:   logrus.InfoLevel,
			}
			err := hook.Fire(entry)
			Expect(err).NotTo(HaveOccurred())

			// Channel should not receive anything
			Consistently(ch).ShouldNot(Receive())
		})
	})
})

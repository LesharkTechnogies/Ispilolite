package queue

import "testing"

func TestQueueRoute(t *testing.T) {
	tests := []struct {
		queue, exchange, key string
	}{
		{JobAssignmentQueue, JobExchange, "job.replay"},
		{NotificationPushQueue, NotificationExchange, "notification.push"},
		{NotificationSMSQueue, NotificationExchange, "notification.sms"},
	}
	for _, test := range tests {
		exchange, key, ok := queueRoute(test.queue)
		if !ok || exchange != test.exchange || key != test.key {
			t.Fatalf("queueRoute(%q) = %q, %q, %v", test.queue, exchange, key, ok)
		}
	}
	if _, _, ok := queueRoute("unknown"); ok {
		t.Fatal("unknown queue was accepted")
	}
}

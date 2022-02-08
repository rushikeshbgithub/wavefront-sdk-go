package senders_test

import (
	"github.com/wavefronthq/wavefront-sdk-go/histogram"
	"github.com/wavefronthq/wavefront-sdk-go/senders"
	"io"
	"net"
	"os"
	"testing"
	"time"
)

func netcat(addr string, keepopen bool, ch chan bool) {
	laddr, _ := net.ResolveTCPAddr("tcp", addr)
	lis, _ := net.ListenTCP("tcp", laddr)
	ch <- true
	for loop := true; loop; loop = keepopen {
		conn, _ := lis.Accept()
		io.Copy(os.Stdout, conn)
	}
	lis.Close()
}

func TestProxySends(t *testing.T) {

	getConnection(t)

	proxyCfg := &senders.ProxyConfiguration{
		Host:                 "localhost",
		MetricsPort:          30000,
		DistributionPort:     40000,
		TracingPort:          50000,
		FlushIntervalSeconds: 10,
	}

	var err error
	var proxy senders.Sender
	if proxy, err = senders.NewProxySender(proxyCfg); err != nil {
		t.Error("Failed Creating Sender", err)
	}

	verifyResults(t, err, proxy)

	proxy.Flush()
	proxy.Close()
	if proxy.GetFailureCount() > 0 {
		t.Error("FailureCount =", proxy.GetFailureCount())
	}
}

func getConnection(t *testing.T) {
	c1 := make(chan bool)
	c2 := make(chan bool)
	c3 := make(chan bool)

	go netcat("localhost:30000", false, c1)
	go netcat("localhost:40000", false, c2)
	go netcat("localhost:50000", false, c3)

	for i := 0; i < 5; i++ {
		if <-c1 && <-c2 && <-c3 {
			break
		} else if i < 4 {
			time.Sleep(time.Second)
		} else {
			t.Fail()
			t.Logf("Could not get a TCP connection")
		}
	}
}

func TestProxySendsWithTags(t *testing.T) {

	getConnection(t)

	proxyCfg := &senders.ProxyConfiguration{
		Host:                 "localhost",
		MetricsPort:          30000,
		DistributionPort:     40000,
		TracingPort:          50000,
		FlushIntervalSeconds: 10,
		SDKMetricsTags:       map[string]string{"foo": "bar"},
	}

	var err error
	var proxy senders.Sender
	if proxy, err = senders.NewProxySender(proxyCfg); err != nil {
		t.Error("Failed Creating Sender", err)
	}

	verifyResults(t, err, proxy)

	proxy.Flush()
	proxy.Close()
	if proxy.GetFailureCount() > 0 {
		t.Error("FailureCount =", proxy.GetFailureCount())
	}
}

func verifyResults(t *testing.T, err error, proxy senders.Sender) {
	if err = proxy.SendMetric("new-york.power.usage", 42422.0, 0, "go_test", map[string]string{"env": "test"}); err != nil {
		t.Error("Failed SendMetric", err)
	}
	if err = proxy.SendDeltaCounter("lambda.thumbnail.generate", 10.0, "thumbnail_service", map[string]string{"format": "jpeg"}); err != nil {
		t.Error("Failed SendDeltaCounter", err)
	}

	centroids := []histogram.Centroid{
		{Value: 30.0, Count: 20},
		{Value: 5.1, Count: 10},
	}

	hgs := map[histogram.Granularity]bool{
		histogram.MINUTE: true,
		histogram.HOUR:   true,
		histogram.DAY:    true,
	}

	if err = proxy.SendDistribution("request.latency", centroids, hgs, 0, "appServer1", map[string]string{"region": "us-west"}); err != nil {
		t.Error("Failed SendDistribution", err)
	}

	if err = proxy.SendSpan("getAllUsers", 0, 343500, "localhost",
		"7b3bf470-9456-11e8-9eb6-529269fb1459", "0313bafe-9457-11e8-9eb6-529269fb1459",
		[]string{"2f64e538-9457-11e8-9eb6-529269fb1459"}, nil,
		[]senders.SpanTag{
			{Key: "application", Value: "Wavefront"},
			{Key: "http.method", Value: "GET"},
		},
		nil); err != nil {
		t.Error("Failed SendSpan", err)
	}
}

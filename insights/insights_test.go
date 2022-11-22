package insights

import (
	"reflect"
	"testing"
	"time"

	"github.com/lightninglabs/faraday/revenue"
	"github.com/lightninglabs/lndclient"
	"github.com/lightningnetwork/lnd/lnwire"
)

// TestGetChannels tests gathering of channel insights from a set of lnrpc
// channels and a revenue report.
func TestGetChannels(t *testing.T) {
	// Make a short channel ID for a channel at height 1000.
	channelHeight1000 := lnwire.ShortChannelID{
		BlockHeight: 1000,
		TxIndex:     1,
		TxPosition:  3,
	}

	// Create an empty revenue report.
	noRevenue := &revenue.Report{
		ChannelPairs: map[string]map[string]revenue.Revenue{},
	}

	// report is a revenue report with the channel opened in block 1000 in
	// it.
	report := &revenue.Report{
		ChannelPairs: map[string]map[string]revenue.Revenue{
			"a:1": {
				"b:1": revenue.Revenue{
					AmountOutgoing: 25,
					AmountIncoming: 10,
					FeesIncoming:   10,
					FeesOutgoing:   10,
				},
				"b:2": revenue.Revenue{
					AmountOutgoing: 0,
					AmountIncoming: 10,
					FeesIncoming:   20,
					FeesOutgoing:   0,
				},
			},
		},
	}

	chanID := channelHeight1000.ToUint64()

	tests := []struct {
		name             string
		channels         []lndclient.ChannelInfo
		currentHeight    uint32
		revenue          *revenue.Report
		expectedInsights []*ChannelInfo
	}{
		{
			name:             "no channels",
			channels:         []lndclient.ChannelInfo{},
			currentHeight:    2000,
			revenue:          noRevenue,
			expectedInsights: []*ChannelInfo{},
		}, {
			name: "one confirmation",
			channels: []lndclient.ChannelInfo{
				{
					ChannelPoint: "a:1",
					LifeTime:     time.Hour,
					Uptime:       time.Hour / 2,
					ChannelID:    chanID,
				},
			},
			currentHeight: 1000,
			revenue:       noRevenue,
			expectedInsights: []*ChannelInfo{
				{
					ChannelPoint:  "a:1",
					MonitoredFor:  time.Hour,
					Uptime:        time.Minute * 30,
					Confirmations: 1,
					Private:       false,
				},
			},
		},
		{
			name: "two confirmations",
			channels: []lndclient.ChannelInfo{
				{
					ChannelPoint: "a:1",
					LifeTime:     time.Hour,
					Uptime:       time.Hour / 2,
					ChannelID:    chanID,
				},
			},
			currentHeight: 1001,
			revenue:       report,
			expectedInsights: []*ChannelInfo{
				{
					ChannelPoint:   "a:1",
					MonitoredFor:   time.Hour,
					Uptime:         time.Minute * 30,
					Confirmations:  2,
					VolumeIncoming: 20,
					VolumeOutgoing: 25,
					FeesEarned:     20,
					Private:        false,
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			insights, err := GetChannels(&Config{
				OpenChannels: func() (
					[]lndclient.ChannelInfo, error) {

					return test.channels, nil
				},
				CurrentHeight: func() (u uint32, e error) {
					return test.currentHeight, nil
				},
				RevenueReport: test.revenue,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(test.expectedInsights) != len(insights) {
				t.Fatalf("expected: %v insights, got: %v",
					len(test.expectedInsights),
					len(insights))
			}

			for i, insight := range test.expectedInsights {
				if !reflect.DeepEqual(insights[i], insight) {
					t.Fatalf("expected: %v, got: %v",
						insight, insights[i])
				}
			}
		})
	}
}

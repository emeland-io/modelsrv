package matchers

import (
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/types"
	"go.emeland.io/modelsrv/pkg/events"
)

func MatchSubscriberUrl(expectedUrl string) types.GomegaMatcher {
	return WithTransform(func(subscriber events.Subscriber) string {
		return subscriber.GetURL()
	}, Equal(expectedUrl))
}

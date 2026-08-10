package main

import (
	"errors"
	"testing"

	"go.emeland.io/modelsrv/pkg/mocks"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestParseCommaSeparatedList(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "empty", raw: "", want: nil},
		{name: "whitespace", raw: "  ,  ", want: nil},
		{name: "single", raw: "http://a/api", want: []string{"http://a/api"}},
		{name: "two", raw: "http://a/api,http://b/api", want: []string{"http://a/api", "http://b/api"}},
		{name: "trim", raw: " http://a/api , http://b/api ", want: []string{"http://a/api", "http://b/api"}},
		{name: "skip empties", raw: "http://a/api,,http://b/api", want: []string{"http://a/api", "http://b/api"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseCommaSeparatedList(tc.raw)
			if len(got) != len(tc.want) {
				t.Fatalf("len=%d want %d (%v)", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("[%d]=%q want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestIsValidSubscriberURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		url  string
		want bool
	}{
		{url: "http://localhost:8080/api", want: true},
		{url: "https://modelsrv.example.com/api/", want: true},
		{url: "", want: false},
		{url: "not-a-url", want: false},
		{url: "ftp://host/api", want: false},
		{url: "http://", want: false},
		{url: "/api", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.url, func(t *testing.T) {
			t.Parallel()
			if got := isValidSubscriberURL(tc.url); got != tc.want {
				t.Fatalf("isValidSubscriberURL(%q)=%v want %v", tc.url, got, tc.want)
			}
		})
	}
}

func TestRegisterStartupSubscribers(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	em := mocks.NewMockEventManager(ctrl)

	validA := "http://replica:8080/api"
	validB := "http://sensor:8081/api"
	invalid := "not-a-url"
	failing := "http://fail.example/api"

	em.EXPECT().AddSubscriber(validA).Return(nil)
	em.EXPECT().AddSubscriber(failing).Return(errors.New("boom"))
	em.EXPECT().AddSubscriber(validB).Return(nil)

	registerStartupSubscribers(em, []string{validA, invalid, failing, validB}, zap.NewNop().Sugar())
}

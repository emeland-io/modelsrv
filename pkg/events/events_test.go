/*
Copyright 2025 Lutz Behnke <lutz.behnke@gmx.de>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package events_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
)

func TestEventManager(t *testing.T) {
	ctx := context.Background()
	em, err := events.NewEventManager()
	assert.NoError(t, err)

	// Test sequence ID management
	seqId, err := em.GetCurrentSequenceId(ctx)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), seqId)

	err = em.IncrementSequenceId(ctx)
	assert.NoError(t, err)

	seqId, err = em.GetCurrentSequenceId(ctx)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), seqId)

	// Test subscriber management
	err = em.AddSubscriber("http://example.com/sink")
	assert.NoError(t, err)

	subscribers := em.GetSubscribers()
	assert.Contains(t, subscribers, "http://example.com/sink")
}

func TestResourceType(t *testing.T) {

	// use AnnotationsResource as a canary to detect if the resource type list has changed, as this is last in the string constant array.
	assert.Equal(t, "Annotations", events.AnnotationsResource.String())

	assert.Equal(t, events.AnnotationsResource, events.ParseResourceType("Annotations"))

}

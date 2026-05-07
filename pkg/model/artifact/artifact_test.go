package artifact_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/artifact"
	"go.emeland.io/modelsrv/pkg/model/common"
)

func TestArtifactBasic(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	assert.NoError(t, err)

	id := uuid.New()
	a := artifact.NewArtifact(m.GetSink(), id)

	a.SetDisplayName("myapp-1.0.tar.gz")
	assert.Equal(t, "myapp-1.0.tar.gz", a.GetDisplayName())

	a.SetDescription("release archive")
	assert.Equal(t, "release archive", a.GetDescription())

	a.SetHash("SHA256:9e9b755d63b36acf30c12a9a3fc379243714c1c6d3dd72861da637f336ebb35b")
	assert.Equal(t, "SHA256:9e9b755d63b36acf30c12a9a3fc379243714c1c6d3dd72861da637f336ebb35b", a.GetHash())

	assert.Nil(t, m.GetArtifactById(id))

	err = m.AddArtifact(a)
	assert.NoError(t, err)
	assert.Same(t, a, m.GetArtifactById(id))

	all, err := m.GetArtifacts()
	assert.NoError(t, err)
	assert.Len(t, all, 1)

	// Update via re-add
	a2 := artifact.NewArtifact(m.GetSink(), id)
	a2.SetDisplayName("myapp-1.1.tar.gz")
	err = m.AddArtifact(a2)
	assert.NoError(t, err)

	// Delete
	err = m.DeleteArtifactById(id)
	assert.NoError(t, err)
	assert.Nil(t, m.GetArtifactById(id))

	err = m.DeleteArtifactById(id)
	assert.ErrorIs(t, err, common.ErrArtifactNotFound)

	eventsList := sink.GetEvents()
	assert.Equal(t, 3, len(eventsList))
	assert.Equal(t, events.ArtifactResource, eventsList[0].ResourceType)
	assert.Equal(t, events.CreateOperation, eventsList[0].Operation)
	assert.Equal(t, events.ArtifactResource, eventsList[1].ResourceType)
	assert.Equal(t, events.UpdateOperation, eventsList[1].Operation)
	assert.Equal(t, events.ArtifactResource, eventsList[2].ResourceType)
	assert.Equal(t, events.DeleteOperation, eventsList[2].Operation)
}

func TestArtifactInstanceBasic(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	assert.NoError(t, err)

	artId := uuid.New()
	art := artifact.NewArtifact(m.GetSink(), artId)
	art.SetDisplayName("base artifact")
	assert.NoError(t, m.AddArtifact(art))

	id := uuid.New()
	ai := artifact.NewArtifactInstance(m.GetSink(), id)
	ai.SetDisplayName("mirror-copy")
	ai.SetDescription("CDN mirror")
	ai.SetArtifactRef(&artifact.ArtifactRef{ArtifactId: artId, Artifact: art})

	assert.Nil(t, m.GetArtifactInstanceById(id))

	err = m.AddArtifactInstance(ai)
	assert.NoError(t, err)
	assert.Same(t, ai, m.GetArtifactInstanceById(id))

	ref := ai.GetArtifactRef()
	assert.NotNil(t, ref)
	assert.Equal(t, artId, ref.ArtifactId)
	assert.Same(t, art, ref.Artifact)

	all, err := m.GetArtifactInstances()
	assert.NoError(t, err)
	assert.Len(t, all, 1)

	err = m.DeleteArtifactInstanceById(id)
	assert.NoError(t, err)
	assert.Nil(t, m.GetArtifactInstanceById(id))

	err = m.DeleteArtifactInstanceById(id)
	assert.ErrorIs(t, err, common.ErrArtifactInstanceNotFound)
}

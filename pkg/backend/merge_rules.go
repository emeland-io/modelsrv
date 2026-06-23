package backend

import (
	"log"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model"
	mdlmergerule "go.emeland.io/modelsrv/pkg/model/mergerule"
)

// mergeRuleNamespace is the UUID namespace for deterministic merge-rule ids.
var mergeRuleNamespace = uuid.MustParse("8b4e3d2c-1a0f-4e5d-9c8b-7a6f5e4d3c2b")

func mergeRuleID(name string) uuid.UUID {
	return uuid.NewSHA1(mergeRuleNamespace, []byte(name))
}

func registerMergeRules(m model.Model) {
	rules := []struct {
		id          uuid.UUID
		displayName string
		description string
	}{
		{
			id:          mergeRuleID("finding-upsert-by-subject-and-kind"),
			displayName: "Finding upsert by subject and kind",
			description: "When the same finding subject and kind is detected again, the existing finding is replaced in place rather than duplicated.",
		},
		{
			id:          mergeRuleID("resource-upsert-by-id"),
			displayName: "Resource upsert by id",
			description: "When a resource with an existing id is added again, the model replaces the stored resource and emits an update event.",
		},
	}

	for _, r := range rules {
		if m.GetMergeRuleById(r.id) != nil {
			continue
		}
		mr := mdlmergerule.NewMergeRule(r.id)
		mr.SetDisplayName(r.displayName)
		mr.SetDescription(r.description)
		if err := m.AddMergeRule(mr); err != nil {
			log.Printf("backend: AddMergeRule id=%s: %v", r.id, err)
		}
	}
}

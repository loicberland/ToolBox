package actions

import "toolBox/pkg/modulecontract"

func Info() modulecontract.ModuleInfo {
	return modulecontract.ModuleInfo{
		ID:          "test-env",
		Name:        "Maquettes de test",
		Description: "Installation et configuration de maquettes de test",
		Actions:     Actions(),
	}
}

func Actions() []modulecontract.ModuleAction {
	return []modulecontract.ModuleAction{
		{ID: "init-config", Name: "Initialiser la configuration", Description: "Cree un fichier de configuration local"},
		{ID: "validate", Name: "Valider", Description: "Verifie la configuration courante"},
	}
}

func Run(actionID string) modulecontract.ActionResponse {
	return modulecontract.ActionResponse{
		ModuleID: "test-env",
		ActionID: actionID,
		Status:   "done",
		Output: map[string]any{
			"message": "test-env action skeleton executed",
		},
	}
}

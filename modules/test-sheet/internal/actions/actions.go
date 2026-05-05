package actions

import "toolBox/pkg/modulecontract"

func Info() modulecontract.ModuleInfo {
	return modulecontract.ModuleInfo{
		ID:          "test-sheet",
		Name:        "Fiches de test",
		Description: "Creation et traitement de fiches de test",
		Actions:     Actions(),
	}
}

func Actions() []modulecontract.ModuleAction {
	return []modulecontract.ModuleAction{
		{ID: "init-db", Name: "Initialiser la base", Description: "Prepare la base SQLite du module"},
		{ID: "list", Name: "Lister", Description: "Retourne les fiches disponibles"},
	}
}

func Run(actionID string) modulecontract.ActionResponse {
	return modulecontract.ActionResponse{
		ModuleID: "test-sheet",
		ActionID: actionID,
		Status:   "done",
		Output: map[string]any{
			"message": "test-sheet action skeleton executed",
		},
	}
}

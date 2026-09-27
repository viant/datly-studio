package store_download_budget

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for budget.
type BudgetComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"budget,path=/_studio/report-resource-files-store/download-budget,method=GET,connector=studio,view=budget,internal=true\" routeName:\"budget\" caseFormat:\"lc\""
}

// BudgetDatlyType keeps the public component type linked for blank-import discovery.
func BudgetDatlyType() reflect.Type { return reflect.TypeOf((*BudgetComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var BudgetDatly = new(BudgetComponent)
var BudgetDatlyLinkedType = BudgetDatlyType()

func (BudgetComponent) EmbedFS() *embed.FS {
	return &BudgetDatlyResources
}

func (BudgetComponent) EmbedNamespace() string {
	return BudgetDatlyResourceNamespace
}

// studio-openapi exports the public Datly SDK contract for client generation.
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/sdk"
	authreader "github.com/viant/datly-studio/studio/auth/reader"
	predicatecreate "github.com/viant/datly-studio/studio/authorization_predicates/create"
	predicatedelete "github.com/viant/datly-studio/studio/authorization_predicates/delete"
	predicateget "github.com/viant/datly-studio/studio/authorization_predicates/get"
	predicatelist "github.com/viant/datly-studio/studio/authorization_predicates/list"
	predicatetypes "github.com/viant/datly-studio/studio/authorization_predicates/types"
	predicateupdate "github.com/viant/datly-studio/studio/authorization_predicates/update"
	connectoractivate "github.com/viant/datly-studio/studio/connectors/activate"
	connectorcreate "github.com/viant/datly-studio/studio/connectors/create"
	connectordelete "github.com/viant/datly-studio/studio/connectors/delete"
	connectordisable "github.com/viant/datly-studio/studio/connectors/disable"
	connectorget "github.com/viant/datly-studio/studio/connectors/get"
	connectors "github.com/viant/datly-studio/studio/connectors/reader"
	connectorschemas "github.com/viant/datly-studio/studio/connectors/schemas"
	connectortable "github.com/viant/datly-studio/studio/connectors/table"
	connectortables "github.com/viant/datly-studio/studio/connectors/tables"
	connectortest "github.com/viant/datly-studio/studio/connectors/test"
	connectortestsql "github.com/viant/datly-studio/studio/connectors/test_sql"
	connectorupdate "github.com/viant/datly-studio/studio/connectors/update"
	"github.com/viant/datly-studio/studio/host"
	namespacecreate "github.com/viant/datly-studio/studio/namespaces/create"
	namespacedelete "github.com/viant/datly-studio/studio/namespaces/delete"
	namespaceget "github.com/viant/datly-studio/studio/namespaces/get"
	namespaces "github.com/viant/datly-studio/studio/namespaces/reader"
	namespaceupdate "github.com/viant/datly-studio/studio/namespaces/update"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	previewexecute "github.com/viant/datly-studio/studio/preview/execute"
	acldelete "github.com/viant/datly-studio/studio/report_acl/delete"
	acl "github.com/viant/datly-studio/studio/report_acl/reader"
	aclupsert "github.com/viant/datly-studio/studio/report_acl/upsert"
	publicationevents "github.com/viant/datly-studio/studio/report_publication_events/list"
	publicationget "github.com/viant/datly-studio/studio/report_publications/get"
	publicationmutate "github.com/viant/datly-studio/studio/report_publications/mutate"
	versionapply "github.com/viant/datly-studio/studio/report_versions/apply"
	versionbuilder "github.com/viant/datly-studio/studio/report_versions/builder"
	versionclone "github.com/viant/datly-studio/studio/report_versions/clone"
	versioncreate "github.com/viant/datly-studio/studio/report_versions/create"
	versiondescriptor "github.com/viant/datly-studio/studio/report_versions/descriptor"
	versiondownload "github.com/viant/datly-studio/studio/report_versions/download"
	versionexport "github.com/viant/datly-studio/studio/report_versions/export_dql"
	versionget "github.com/viant/datly-studio/studio/report_versions/get"
	versioninspect "github.com/viant/datly-studio/studio/report_versions/inspect"
	versionlist "github.com/viant/datly-studio/studio/report_versions/list"
	versionarchive "github.com/viant/datly-studio/studio/report_versions/load_archive"
	versionload "github.com/viant/datly-studio/studio/report_versions/load_dql"
	versiontestcompose "github.com/viant/datly-studio/studio/report_versions/test_compose"
	versiontestrelation "github.com/viant/datly-studio/studio/report_versions/test_relation"
	versiontestview "github.com/viant/datly-studio/studio/report_versions/test_view"
	versionvalidate "github.com/viant/datly-studio/studio/report_versions/validate"
	versionwarmup "github.com/viant/datly-studio/studio/report_versions/warmup"
	warmupget "github.com/viant/datly-studio/studio/report_warmup_runs/get"
	warmuplist "github.com/viant/datly-studio/studio/report_warmup_runs/list"
	catalogpredicate "github.com/viant/datly-studio/studio/reports/catalogpredicate"
	reportcreate "github.com/viant/datly-studio/studio/reports/create"
	reportget "github.com/viant/datly-studio/studio/reports/get"
	reports "github.com/viant/datly-studio/studio/reports/reader"
	reportupdate "github.com/viant/datly-studio/studio/reports/update"
	resourceaccess "github.com/viant/datly-studio/studio/resource_policy/access"
	resourceget "github.com/viant/datly-studio/studio/resources/get"
	resourcemutate "github.com/viant/datly-studio/studio/resources/mutate"
	runtimestatus "github.com/viant/datly-studio/studio/runtime/status"
	"github.com/viant/datly/bootstrap"
	"github.com/viant/datly/gateway/openapi"
	"github.com/viant/datly/gateway/openapi/openapi3"
	runtimeauth "github.com/viant/datly/runtime/auth"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/tag"
	"github.com/viant/datly/typecatalog"
	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt/verifier"
	"github.com/viant/x"
	xcodec "github.com/viant/xdatly/codec"
)

func main() {
	output := flag.String("out", "sdk/openapi/studio.json", "generated Studio SDK OpenAPI document")
	flag.Parse()
	if err := run(context.Background(), *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, output string) error {
	resources := resource.New()
	if err := resources.Register(acl.AclDatlyResourceNamespace, acl.AclDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(connectors.ConnectorDatlyResourceNamespace, connectors.ConnectorDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(connectorget.ConnectorDatlyResourceNamespace, connectorget.ConnectorDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(connectorcreate.ConnectorDatlyResourceNamespace, connectorcreate.ConnectorDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(namespaces.NamespaceDatlyResourceNamespace, namespaces.NamespaceDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(namespaceget.NamespaceDatlyResourceNamespace, namespaceget.NamespaceDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(namespacecreate.NamespaceDatlyResourceNamespace, namespacecreate.NamespaceDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(authreader.ContextDatlyResourceNamespace, authreader.ContextDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(reports.ReportDatlyResourceNamespace, reports.ReportDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(reportget.ReportDatlyResourceNamespace, reportget.ReportDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(publicationget.PublicationDatlyResourceNamespace, publicationget.PublicationDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(publicationevents.EventDatlyResourceNamespace, publicationevents.EventDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(versionget.VersionDatlyResourceNamespace, versionget.VersionDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(versionexport.VersionDatlyResourceNamespace, versionexport.VersionDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(versiondescriptor.VersionDatlyResourceNamespace, versiondescriptor.VersionDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(versionlist.VersionDatlyResourceNamespace, versionlist.VersionDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(warmupget.WarmupRunDatlyResourceNamespace, warmupget.WarmupRunDatlyResources); err != nil {
		return err
	}
	predicates, err := (host.Config{PredicatePackages: []predicatecatalog.Package{{
		Alias: "catalogpredicate", Path: "github.com/viant/datly-studio/studio/reports/catalogpredicate",
		Types: []reflect.Type{reflect.TypeFor[catalogpredicate.ReportCatalogRead]()},
	}}}).PredicateCatalog()
	if err != nil {
		return err
	}
	types, err := predicates.RuntimeTypes()
	if err != nil {
		return err
	}
	for _, value := range []any{sdk.WarmupRun{}, sdk.WarmupTarget{}, sdk.Diagnostic{}, sdk.ReportVersion{}, sdk.ReportCapabilities{}, sdk.AuthorizationPredicateType{}, sdk.AuthorizationPredicate{}, sdk.SchemaCatalogInput{}, sdk.DatabaseSchema{}, sdk.TableCatalogInput{}, sdk.TableDetailInput{}, sdk.DatabaseTable{}, sdk.DatabaseColumn{}, sdk.SQLTestInput{}, sdk.PreviewInput{}, sdk.ExecutionEvidence{}, sdk.ViewTestInput{}, sdk.RelationKeyEvidence{}, sdk.CubeComposeTestInput{}} {
		if err := types.Register(typecatalog.TypeOriginPackage, x.NewType(reflect.TypeOf(value))); err != nil {
			return err
		}
	}
	// OpenAPI only treats Authorization as bearer security when the compiled
	// codec is a real verifier. A throwaway key keeps generation offline while
	// exercising the same verified-JWT contract as the serving Datly runtime.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return err
	}
	codec, err := runtimeauth.New(ctx, &runtimeauth.Config{JWTValidator: &verifier.Config{RSA: []*scy.Resource{{
		URL: "studio-openapi-ephemeral", Data: pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER}),
	}}}})
	if err != nil {
		return err
	}
	auth, err := compile(reflect.TypeFor[authreader.ContextComponent](), reflect.TypeFor[authreader.Input](),
		reflect.TypeFor[authreader.Output](), resources, types, codec)
	if err != nil {
		return err
	}
	aclList, err := compile(reflect.TypeFor[acl.AclComponent](), reflect.TypeFor[acl.Input](),
		reflect.TypeFor[acl.Output](), resources, types, codec)
	if err != nil {
		return err
	}
	aclDeleteHandler, err := (acldelete.Component{}).DatlyHandler("NewACLDelete")()
	if err != nil {
		return err
	}
	aclDelete, err := compile(reflect.TypeFor[acldelete.Component](), reflect.TypeFor[acldelete.Input](),
		reflect.TypeFor[acldelete.Output](), resources, types, codec, aclDeleteHandler)
	if err != nil {
		return err
	}
	aclUpsertHandler, err := (aclupsert.Component{}).DatlyHandler("NewACLUpsert")()
	if err != nil {
		return err
	}
	aclUpsert, err := compile(reflect.TypeFor[aclupsert.Component](), reflect.TypeFor[aclupsert.Input](),
		reflect.TypeFor[aclupsert.Output](), resources, types, codec, aclUpsertHandler)
	if err != nil {
		return err
	}
	connectorList, err := compile(reflect.TypeFor[connectors.ConnectorComponent](), reflect.TypeFor[connectors.Input](),
		reflect.TypeFor[connectors.Output](), resources, types, codec)
	if err != nil {
		return err
	}
	connectorOne, err := compile(reflect.TypeFor[connectorget.ConnectorComponent](), reflect.TypeFor[connectorget.ConnectorGetInput](),
		reflect.TypeFor[connectorget.ConnectorGetOutput](), resources, types, codec)
	if err != nil {
		return err
	}
	connectorCreate, err := compile(reflect.TypeFor[connectorcreate.ConnectorComponent](), reflect.TypeFor[connectorcreate.ConnectorCreateInput](),
		reflect.TypeFor[connectorcreate.ConnectorCreateOutput](), resources, types, codec)
	if err != nil {
		return err
	}
	connectorSchemasHandler, err := (connectorschemas.Component{}).DatlyHandler("NewSchemas")()
	if err != nil {
		return err
	}
	connectorSchemas, err := compile(reflect.TypeFor[connectorschemas.Component](), reflect.TypeFor[connectorschemas.Input](),
		reflect.TypeFor[connectorschemas.Output](), resources, types, codec, connectorSchemasHandler)
	if err != nil {
		return err
	}
	connectorTablesHandler, err := (connectortables.Component{}).DatlyHandler("NewTables")()
	if err != nil {
		return err
	}
	connectorTables, err := compile(reflect.TypeFor[connectortables.Component](), reflect.TypeFor[connectortables.Input](),
		reflect.TypeFor[connectortables.Output](), resources, types, codec, connectorTablesHandler)
	if err != nil {
		return err
	}
	connectorTableHandler, err := (connectortable.Component{}).DatlyHandler("NewTable")()
	if err != nil {
		return err
	}
	connectorTable, err := compile(reflect.TypeFor[connectortable.Component](), reflect.TypeFor[connectortable.Input](),
		reflect.TypeFor[connectortable.Output](), resources, types, codec, connectorTableHandler)
	if err != nil {
		return err
	}
	connectorTestHandler, err := (connectortest.Component{}).DatlyHandler("NewTest")()
	if err != nil {
		return err
	}
	connectorTest, err := compile(reflect.TypeFor[connectortest.Component](), reflect.TypeFor[connectortest.Input](),
		reflect.TypeFor[connectortest.Output](), resources, types, codec, connectorTestHandler)
	if err != nil {
		return err
	}
	connectorTestSQLHandler, err := (connectortestsql.Component{}).DatlyHandler("NewTestSQL")()
	if err != nil {
		return err
	}
	connectorTestSQL, err := compile(reflect.TypeFor[connectortestsql.Component](), reflect.TypeFor[connectortestsql.Input](),
		reflect.TypeFor[connectortestsql.Output](), resources, types, codec, connectorTestSQLHandler)
	if err != nil {
		return err
	}
	previewHandler, err := (previewexecute.Component{}).DatlyHandler("NewExecute")()
	if err != nil {
		return err
	}
	previewExecution, err := compile(reflect.TypeFor[previewexecute.Component](), reflect.TypeFor[previewexecute.Input](),
		reflect.TypeFor[previewexecute.Output](), resources, types, codec, previewHandler)
	if err != nil {
		return err
	}
	connectorDeleteHandler, err := (connectordelete.Component{}).DatlyHandler("NewConnectorDelete")()
	if err != nil {
		return err
	}
	connectorDelete, err := compile(reflect.TypeFor[connectordelete.Component](), reflect.TypeFor[connectordelete.Input](),
		reflect.TypeFor[connectordelete.Output](), resources, types, codec, connectorDeleteHandler)
	if err != nil {
		return err
	}
	connectorActivateHandler, err := (connectoractivate.Component{}).DatlyHandler("NewConnectorActivate")()
	if err != nil {
		return err
	}
	connectorActivate, err := compile(reflect.TypeFor[connectoractivate.Component](), reflect.TypeFor[connectoractivate.Input](),
		reflect.TypeFor[connectoractivate.Output](), resources, types, codec, connectorActivateHandler)
	if err != nil {
		return err
	}
	connectorDisableHandler, err := (connectordisable.Component{}).DatlyHandler("NewConnectorDisable")()
	if err != nil {
		return err
	}
	connectorDisable, err := compile(reflect.TypeFor[connectordisable.Component](), reflect.TypeFor[connectordisable.Input](),
		reflect.TypeFor[connectordisable.Output](), resources, types, codec, connectorDisableHandler)
	if err != nil {
		return err
	}
	connectorUpdateHandler, err := (connectorupdate.Component{}).DatlyHandler("NewConnectorUpdate")()
	if err != nil {
		return err
	}
	connectorUpdate, err := compile(reflect.TypeFor[connectorupdate.Component](), reflect.TypeFor[connectorupdate.Input](),
		reflect.TypeFor[connectorupdate.Output](), resources, types, codec, connectorUpdateHandler)
	if err != nil {
		return err
	}
	namespaceList, err := compile(reflect.TypeFor[namespaces.NamespaceComponent](), reflect.TypeFor[namespaces.NamespaceQueryInput](),
		reflect.TypeFor[namespaces.NamespaceQueryOutput](), resources, types, codec)
	if err != nil {
		return err
	}
	namespaceOne, err := compile(reflect.TypeFor[namespaceget.NamespaceComponent](), reflect.TypeFor[namespaceget.NamespaceGetInput](),
		reflect.TypeFor[namespaceget.NamespaceGetOutput](), resources, types, codec)
	if err != nil {
		return err
	}
	namespaceCreate, err := compile(reflect.TypeFor[namespacecreate.NamespaceComponent](), reflect.TypeFor[namespacecreate.NamespaceCreateInput](),
		reflect.TypeFor[namespacecreate.NamespaceCreateOutput](), resources, types, codec)
	if err != nil {
		return err
	}
	namespaceDeleteHandler, err := (namespacedelete.Component{}).DatlyHandler("NewNamespaceDelete")()
	if err != nil {
		return err
	}
	namespaceDelete, err := compile(reflect.TypeFor[namespacedelete.Component](), reflect.TypeFor[namespacedelete.Input](),
		reflect.TypeFor[namespacedelete.Output](), resources, types, codec, namespaceDeleteHandler)
	if err != nil {
		return err
	}
	namespaceUpdateHandler, err := (namespaceupdate.Component{}).DatlyHandler("NewNamespaceUpdate")()
	if err != nil {
		return err
	}
	namespaceUpdate, err := compile(reflect.TypeFor[namespaceupdate.Component](), reflect.TypeFor[namespaceupdate.Input](),
		reflect.TypeFor[namespaceupdate.Output](), resources, types, codec, namespaceUpdateHandler)
	if err != nil {
		return err
	}
	reportList, err := compile(reflect.TypeFor[reports.ReportComponent](), reflect.TypeFor[reports.Input](),
		reflect.TypeFor[reports.Output](), resources, types, codec)
	if err != nil {
		return err
	}
	reportOne, err := compile(reflect.TypeFor[reportget.ReportComponent](), reflect.TypeFor[reportget.ReportGetInput](),
		reflect.TypeFor[reportget.ReportGetOutput](), resources, types, codec)
	if err != nil {
		return err
	}
	reportCreateHandler, err := (reportcreate.Component{}).DatlyHandler("NewReportCreate")()
	if err != nil {
		return err
	}
	reportCreate, err := compile(reflect.TypeFor[reportcreate.Component](), reflect.TypeFor[reportcreate.Input](),
		reflect.TypeFor[reportcreate.Output](), resources, types, codec, reportCreateHandler)
	if err != nil {
		return err
	}
	reportUpdateHandler, err := (reportupdate.Component{}).DatlyHandler("NewReportUpdate")()
	if err != nil {
		return err
	}
	reportUpdate, err := compile(reflect.TypeFor[reportupdate.Component](), reflect.TypeFor[reportupdate.Input](),
		reflect.TypeFor[reportupdate.Output](), resources, types, codec, reportUpdateHandler)
	if err != nil {
		return err
	}
	publicationOne, err := compile(reflect.TypeFor[publicationget.PublicationComponent](), reflect.TypeFor[publicationget.PublicationGetInput](),
		reflect.TypeFor[publicationget.PublicationGetOutput](), resources, types, codec)
	if err != nil {
		return err
	}
	publicationHistory, err := compile(reflect.TypeFor[publicationevents.EventComponent](), reflect.TypeFor[publicationevents.PublicationEventsListInput](),
		reflect.TypeFor[publicationevents.PublicationEventsListOutput](), resources, types, codec)
	if err != nil {
		return err
	}
	predicateTypesHandler, err := (predicatetypes.Component{}).DatlyHandler("NewTypes")()
	if err != nil {
		return err
	}
	predicateTypes, err := compile(reflect.TypeFor[predicatetypes.Component](), reflect.TypeFor[predicatetypes.Input](),
		reflect.TypeFor[predicatetypes.Output](), resources, types, codec, predicateTypesHandler)
	if err != nil {
		return err
	}
	predicateGetHandler, err := (predicateget.Component{}).DatlyHandler("NewGet")()
	if err != nil {
		return err
	}
	predicateGet, err := compile(reflect.TypeFor[predicateget.Component](), reflect.TypeFor[predicateget.Input](),
		reflect.TypeFor[predicateget.Output](), resources, types, codec, predicateGetHandler)
	if err != nil {
		return err
	}
	predicateCreateHandler, err := (predicatecreate.Component{}).DatlyHandler("NewCreate")()
	if err != nil {
		return err
	}
	predicateCreate, err := compile(reflect.TypeFor[predicatecreate.Component](), reflect.TypeFor[predicatecreate.Input](),
		reflect.TypeFor[predicateget.Output](), resources, types, codec, predicateCreateHandler)
	if err != nil {
		return err
	}
	predicateUpdateHandler, err := (predicateupdate.Component{}).DatlyHandler("NewUpdate")()
	if err != nil {
		return err
	}
	predicateUpdate, err := compile(reflect.TypeFor[predicateupdate.Component](), reflect.TypeFor[predicateupdate.Input](),
		reflect.TypeFor[predicateget.Output](), resources, types, codec, predicateUpdateHandler)
	if err != nil {
		return err
	}
	predicateDeleteHandler, err := (predicatedelete.Component{}).DatlyHandler("NewDelete")()
	if err != nil {
		return err
	}
	predicateDelete, err := compile(reflect.TypeFor[predicatedelete.Component](), reflect.TypeFor[predicatedelete.Input](),
		reflect.TypeFor[predicatedelete.Output](), resources, types, codec, predicateDeleteHandler)
	if err != nil {
		return err
	}
	predicateListHandler, err := (predicatelist.Component{}).DatlyHandler("NewList")()
	if err != nil {
		return err
	}
	predicateList, err := compile(reflect.TypeFor[predicatelist.Component](), reflect.TypeFor[predicatelist.Input](),
		reflect.TypeFor[predicatelist.Output](), resources, types, codec, predicateListHandler)
	if err != nil {
		return err
	}
	versionOne, err := compile(reflect.TypeFor[versionget.VersionComponent](), reflect.TypeFor[versionget.VersionGetInput](),
		reflect.TypeFor[versionget.VersionGetOutput](), resources, types, codec)
	if err != nil {
		return err
	}
	versionCreateHandler, err := (versioncreate.Component{}).DatlyHandler("NewVersionCreate")()
	if err != nil {
		return err
	}
	versionCloneHandler, err := (versionclone.Component{}).DatlyHandler("NewClone")()
	if err != nil {
		return err
	}
	versionClone, err := compile(reflect.TypeFor[versionclone.Component](), reflect.TypeFor[versionclone.Input](), reflect.TypeFor[versionclone.Output](), resources, types, codec, versionCloneHandler)
	if err != nil {
		return err
	}
	versionCreate, err := compile(reflect.TypeFor[versioncreate.Component](), reflect.TypeFor[versioncreate.Input](),
		reflect.TypeFor[versioncreate.Output](), resources, types, codec, versionCreateHandler)
	if err != nil {
		return err
	}
	versionLoadHandler, err := (versionload.Component{}).DatlyHandler("NewLoadDQL")()
	if err != nil {
		return err
	}
	versionLoad, err := compile(reflect.TypeFor[versionload.Component](), reflect.TypeFor[versionload.Input](),
		reflect.TypeFor[versionload.Output](), resources, types, codec, versionLoadHandler)
	if err != nil {
		return err
	}
	versionArchiveHandler, err := (versionarchive.Component{}).DatlyHandler("NewLoadArchive")()
	if err != nil {
		return err
	}
	versionArchive, err := compile(reflect.TypeFor[versionarchive.Component](), reflect.TypeFor[versionarchive.Input](),
		reflect.TypeFor[versionarchive.Output](), resources, types, codec, versionArchiveHandler)
	if err != nil {
		return err
	}
	versionInspectHandler, err := (versioninspect.Component{}).DatlyHandler("NewVersionInspect")()
	if err != nil {
		return err
	}
	versionInspect, err := compile(reflect.TypeFor[versioninspect.Component](), reflect.TypeFor[versioninspect.Input](),
		reflect.TypeFor[versioninspect.Output](), resources, types, codec, versionInspectHandler)
	if err != nil {
		return err
	}
	versionApplyHandler, err := (versionapply.Component{}).DatlyHandler("NewVersionApply")()
	if err != nil {
		return err
	}
	versionApply, err := compile(reflect.TypeFor[versionapply.Component](), reflect.TypeFor[versionapply.Input](),
		reflect.TypeFor[versionapply.Output](), resources, types, codec, versionApplyHandler)
	if err != nil {
		return err
	}
	versionBuilderHandler, err := (versionbuilder.Component{}).DatlyHandler("NewReaderBuilder")()
	if err != nil {
		return err
	}
	versionBuilder, err := compile(reflect.TypeFor[versionbuilder.Component](), reflect.TypeFor[versionbuilder.Input](),
		reflect.TypeFor[versionbuilder.Output](), resources, types, codec, versionBuilderHandler)
	if err != nil {
		return err
	}
	versionValidateHandler, err := (versionvalidate.Component{}).DatlyHandler("NewValidate")()
	if err != nil {
		return err
	}
	versionValidate, err := compile(reflect.TypeFor[versionvalidate.Component](), reflect.TypeFor[versionvalidate.Input](),
		reflect.TypeFor[versionvalidate.Output](), resources, types, codec, versionValidateHandler)
	if err != nil {
		return err
	}
	versionTestViewHandler, err := (versiontestview.Component{}).DatlyHandler("NewTestView")()
	if err != nil {
		return err
	}
	versionTestView, err := compile(reflect.TypeFor[versiontestview.Component](), reflect.TypeFor[versiontestview.Input](),
		reflect.TypeFor[versiontestview.Output](), resources, types, codec, versionTestViewHandler)
	if err != nil {
		return err
	}
	versionTestRelationHandler, err := (versiontestrelation.Component{}).DatlyHandler("NewTestRelation")()
	if err != nil {
		return err
	}
	versionTestRelation, err := compile(reflect.TypeFor[versiontestrelation.Component](), reflect.TypeFor[versiontestrelation.Input](),
		reflect.TypeFor[versiontestrelation.Output](), resources, types, codec, versionTestRelationHandler)
	if err != nil {
		return err
	}
	versionTestComposeHandler, err := (versiontestcompose.Component{}).DatlyHandler("NewTestCompose")()
	if err != nil {
		return err
	}
	versionTestCompose, err := compile(reflect.TypeFor[versiontestcompose.Component](), reflect.TypeFor[versiontestcompose.Input](),
		reflect.TypeFor[versiontestcompose.Output](), resources, types, codec, versionTestComposeHandler)
	if err != nil {
		return err
	}
	versionWarmupHandler, err := (versionwarmup.Component{}).DatlyHandler("NewWarmup")()
	if err != nil {
		return err
	}
	versionWarmup, err := compile(reflect.TypeFor[versionwarmup.Component](), reflect.TypeFor[versionwarmup.Input](),
		reflect.TypeFor[versionwarmup.Output](), resources, types, codec, versionWarmupHandler)
	if err != nil {
		return err
	}
	versionExport, err := compile(reflect.TypeFor[versionexport.VersionComponent](), reflect.TypeFor[versionexport.VersionExportInput](),
		reflect.TypeFor[versionexport.VersionExportOutput](), resources, types, codec)
	if err != nil {
		return err
	}
	versionDescriptor, err := compile(reflect.TypeFor[versiondescriptor.VersionComponent](), reflect.TypeFor[versiondescriptor.VersionDescriptorInput](),
		reflect.TypeFor[versiondescriptor.VersionDescriptorOutput](), resources, types, codec)
	if err != nil {
		return err
	}
	downloadHandler, err := (versiondownload.Component{}).DatlyHandler("NewDownload")()
	if err != nil {
		return err
	}
	versionDownload, err := compile(reflect.TypeFor[versiondownload.Component](), reflect.TypeFor[versiondownload.Input](),
		reflect.TypeFor[versiondownload.Output](), resources, types, codec, downloadHandler)
	if err != nil {
		return err
	}
	resourceHandler, err := (resourceget.Component{}).DatlyHandler("NewResourceSnapshot")()
	if err != nil {
		return err
	}
	resourceSnapshot, err := compile(reflect.TypeFor[resourceget.Component](), reflect.TypeFor[resourceget.Input](),
		reflect.TypeFor[resourceget.Output](), resources, types, codec, resourceHandler)
	if err != nil {
		return err
	}
	compileResourceMutation := func(holder, input reflect.Type, factory func() (rhandler.TypedHandler, error)) (*registry.RegisteredComponent, error) {
		handler, err := factory()
		if err != nil {
			return nil, err
		}
		return compile(holder, input, reflect.TypeFor[resourcemutate.Output](), resources, types, codec, handler)
	}
	resourceUpsertFile, err := compileResourceMutation(reflect.TypeFor[resourcemutate.FileUpsertComponent](), reflect.TypeFor[resourcemutate.FileUpsertInput](), (resourcemutate.FileUpsertComponent{}).DatlyHandler("NewUpsertFile"))
	if err != nil {
		return err
	}
	resourceDeleteFile, err := compileResourceMutation(reflect.TypeFor[resourcemutate.FileDeleteComponent](), reflect.TypeFor[resourcemutate.FileDeleteInput](), (resourcemutate.FileDeleteComponent{}).DatlyHandler("NewDeleteFile"))
	if err != nil {
		return err
	}
	resourceUpsertFolder, err := compileResourceMutation(reflect.TypeFor[resourcemutate.FolderUpsertComponent](), reflect.TypeFor[resourcemutate.FolderUpsertInput](), (resourcemutate.FolderUpsertComponent{}).DatlyHandler("NewUpsertFolder"))
	if err != nil {
		return err
	}
	resourceDeleteFolder, err := compileResourceMutation(reflect.TypeFor[resourcemutate.FolderDeleteComponent](), reflect.TypeFor[resourcemutate.FolderDeleteInput](), (resourcemutate.FolderDeleteComponent{}).DatlyHandler("NewDeleteFolder"))
	if err != nil {
		return err
	}
	resourceUpsertSkill, err := compileResourceMutation(reflect.TypeFor[resourcemutate.SkillUpsertComponent](), reflect.TypeFor[resourcemutate.SkillUpsertInput](), (resourcemutate.SkillUpsertComponent{}).DatlyHandler("NewUpsertSkill"))
	if err != nil {
		return err
	}
	resourceDeleteSkill, err := compileResourceMutation(reflect.TypeFor[resourcemutate.SkillDeleteComponent](), reflect.TypeFor[resourcemutate.SkillDeleteInput](), (resourcemutate.SkillDeleteComponent{}).DatlyHandler("NewDeleteSkill"))
	if err != nil {
		return err
	}
	versionPage, err := compile(reflect.TypeFor[versionlist.VersionComponent](), reflect.TypeFor[versionlist.VersionListInput](),
		reflect.TypeFor[versionlist.VersionListOutput](), resources, types, codec)
	if err != nil {
		return err
	}
	warmupOne, err := compile(reflect.TypeFor[warmupget.WarmupRunComponent](), reflect.TypeFor[warmupget.WarmupGetInput](),
		reflect.TypeFor[warmupget.WarmupGetOutput](), resources, types, codec)
	if err != nil {
		return err
	}
	warmupListHandler, err := (warmuplist.Component{}).DatlyHandler("NewWarmupList")()
	if err != nil {
		return err
	}
	warmupPage, err := compile(reflect.TypeFor[warmuplist.Component](), reflect.TypeFor[warmuplist.Input](),
		reflect.TypeFor[warmuplist.Output](), resources, types, codec, warmupListHandler)
	if err != nil {
		return err
	}
	runtimeStatusHandler, err := (runtimestatus.Component{}).DatlyHandler("NewStatus")()
	if err != nil {
		return err
	}
	runtimeStatus, err := compile(reflect.TypeFor[runtimestatus.Component](), reflect.TypeFor[runtimestatus.Input](),
		reflect.TypeFor[runtimestatus.Output](), resources, types, codec, runtimeStatusHandler)
	if err != nil {
		return err
	}
	compilePublicationMutation := func(holder, input reflect.Type, factory func() (rhandler.TypedHandler, error)) (*registry.RegisteredComponent, error) {
		handler, err := factory()
		if err != nil {
			return nil, err
		}
		return compile(holder, input, reflect.TypeFor[publicationmutate.Output](), resources, types, codec, handler)
	}
	publicationPublish, err := compilePublicationMutation(reflect.TypeFor[publicationmutate.PublishComponent](), reflect.TypeFor[publicationmutate.VersionInput](), (publicationmutate.PublishComponent{}).DatlyHandler("NewPublish"))
	if err != nil {
		return err
	}
	publicationRollback, err := compilePublicationMutation(reflect.TypeFor[publicationmutate.RollbackComponent](), reflect.TypeFor[publicationmutate.VersionInput](), (publicationmutate.RollbackComponent{}).DatlyHandler("NewRollback"))
	if err != nil {
		return err
	}
	publicationUnpublish, err := compilePublicationMutation(reflect.TypeFor[publicationmutate.UnpublishComponent](), reflect.TypeFor[publicationmutate.UnpublishInput](), (publicationmutate.UnpublishComponent{}).DatlyHandler("NewUnpublish"))
	if err != nil {
		return err
	}
	accessListHandler, err := (resourceaccess.ListComponent{}).DatlyHandler("NewList")()
	if err != nil {
		return err
	}
	accessList, err := compile(reflect.TypeFor[resourceaccess.ListComponent](), reflect.TypeFor[resourceaccess.ListInput](), reflect.TypeFor[resourceaccess.ListOutput](), resources, types, codec, accessListHandler)
	if err != nil {
		return err
	}
	accessGetHandler, err := (resourceaccess.GetComponent{}).DatlyHandler("NewGet")()
	if err != nil {
		return err
	}
	accessGet, err := compile(reflect.TypeFor[resourceaccess.GetComponent](), reflect.TypeFor[resourceaccess.ResourceInput](), reflect.TypeFor[resourceaccess.DocumentOutput](), resources, types, codec, accessGetHandler)
	if err != nil {
		return err
	}
	accessContextHandler, err := (resourceaccess.ContextComponent{}).DatlyHandler("NewContext")()
	if err != nil {
		return err
	}
	accessContext, err := compile(reflect.TypeFor[resourceaccess.ContextComponent](), reflect.TypeFor[resourceaccess.ResourceInput](), reflect.TypeFor[resourceaccess.ContextOutput](), resources, types, codec, accessContextHandler)
	if err != nil {
		return err
	}
	accessReplaceHandler, err := (resourceaccess.ReplaceComponent{}).DatlyHandler("NewReplace")()
	if err != nil {
		return err
	}
	accessReplace, err := compile(reflect.TypeFor[resourceaccess.ReplaceComponent](), reflect.TypeFor[resourceaccess.ReplaceInput](), reflect.TypeFor[resourceaccess.DocumentOutput](), resources, types, codec, accessReplaceHandler)
	if err != nil {
		return err
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{
		Info:       openapi3.Info{Title: "Datly Studio SDK", Version: "1.0.0"},
		Components: []*registry.RegisteredComponent{auth, accessList, accessGet, accessContext, accessReplace, aclList, aclDelete, aclUpsert, predicateTypes, predicateGet, predicateCreate, predicateUpdate, predicateDelete, predicateList, connectorList, connectorOne, connectorCreate, connectorSchemas, connectorTables, connectorTable, connectorTest, connectorTestSQL, previewExecution, connectorActivate, connectorDelete, connectorDisable, connectorUpdate, namespaceList, namespaceOne, namespaceCreate, namespaceDelete, namespaceUpdate, reportList, reportOne, reportCreate, reportUpdate, publicationOne, publicationHistory, publicationPublish, publicationRollback, publicationUnpublish, versionOne, versionCreate, versionClone, versionLoad, versionArchive, versionInspect, versionApply, versionBuilder, versionValidate, versionTestView, versionTestRelation, versionTestCompose, versionWarmup, versionPage, versionExport, versionDescriptor, versionDownload, resourceSnapshot, resourceUpsertFile, resourceDeleteFile, resourceUpsertFolder, resourceDeleteFolder, resourceUpsertSkill, resourceDeleteSkill, warmupOne, warmupPage, runtimeStatus},
		Routes: []spec.RouteRef{
			{Method: "POST", Path: "/v1/studio/sdk/access.list"},
			{Method: "POST", Path: "/v1/studio/sdk/access.context"},
			{Method: "POST", Path: "/v1/studio/sdk/access.get"},
			{Method: "POST", Path: "/v1/studio/sdk/access.replace"},
			{Method: "POST", Path: "/v1/studio/sdk/authorization_predicates.types"},
			{Method: "POST", Path: "/v1/studio/sdk/authorization_predicates.get"},
			{Method: "POST", Path: "/v1/studio/sdk/authorization_predicates.create"},
			{Method: "POST", Path: "/v1/studio/sdk/authorization_predicates.update"},
			{Method: "POST", Path: "/v1/studio/sdk/authorization_predicates.delete"},
			{Method: "POST", Path: "/v1/studio/sdk/authorization_predicates.list"},
			{Method: "POST", Path: "/v1/studio/sdk/acl.list"},
			{Method: "POST", Path: "/v1/studio/sdk/acl.delete"},
			{Method: "POST", Path: "/v1/studio/sdk/acl.upsert"},
			{Method: "POST", Path: "/v1/studio/sdk/connectors.get"},
			{Method: "POST", Path: "/v1/studio/sdk/connectors.create"},
			{Method: "POST", Path: "/v1/studio/sdk/connectors.schemas"},
			{Method: "POST", Path: "/v1/studio/sdk/connectors.tables"},
			{Method: "POST", Path: "/v1/studio/sdk/connectors.table"},
			{Method: "POST", Path: "/v1/studio/sdk/connectors.test"},
			{Method: "POST", Path: "/v1/studio/sdk/connectors.test_sql"},
			{Method: "POST", Path: "/v1/studio/sdk/preview.execute"},
			{Method: "POST", Path: "/v1/studio/sdk/connectors.activate"},
			{Method: "POST", Path: "/v1/studio/sdk/connectors.delete"},
			{Method: "POST", Path: "/v1/studio/sdk/connectors.disable"},
			{Method: "POST", Path: "/v1/studio/sdk/connectors.update"},
			{Method: "POST", Path: "/v1/studio/sdk/connectors.list"},
			{Method: "POST", Path: "/v1/studio/sdk/namespaces.list"},
			{Method: "POST", Path: "/v1/studio/sdk/namespaces.get"},
			{Method: "POST", Path: "/v1/studio/sdk/namespaces.create"},
			{Method: "POST", Path: "/v1/studio/sdk/namespaces.delete"},
			{Method: "POST", Path: "/v1/studio/sdk/namespaces.update"},
			{Method: "POST", Path: "/v1/studio/sdk/components.get"},
			{Method: "POST", Path: "/v1/studio/sdk/components.create"},
			{Method: "POST", Path: "/v1/studio/sdk/components.update"},
			{Method: "POST", Path: "/v1/studio/sdk/components.list"},
			{Method: "POST", Path: "/v1/studio/sdk/publications.get"},
			{Method: "POST", Path: "/v1/studio/sdk/publications.events.list"},
			{Method: "POST", Path: "/v1/studio/sdk/publications.publish"},
			{Method: "POST", Path: "/v1/studio/sdk/publications.rollback"},
			{Method: "POST", Path: "/v1/studio/sdk/publications.unpublish"},
			{Method: "POST", Path: "/v1/studio/sdk/runtime.status"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.get"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.create"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.clone"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.load_dql"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.load_archive"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.inspect"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.apply"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.builder"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.validate"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.test_view"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.test_relation"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.test_compose"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.warmup"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.list"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.export_dql"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.descriptor"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.download"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.warmup_get"},
			{Method: "POST", Path: "/v1/studio/sdk/versions.warmup_list"},
			{Method: "POST", Path: "/v1/studio/sdk/resources.get"},
			{Method: "POST", Path: "/v1/studio/sdk/resources.upsert_file"},
			{Method: "POST", Path: "/v1/studio/sdk/resources.delete_file"},
			{Method: "POST", Path: "/v1/studio/sdk/resources.upsert_folder"},
			{Method: "POST", Path: "/v1/studio/sdk/resources.delete_folder"},
			{Method: "POST", Path: "/v1/studio/sdk/resources.upsert_skill"},
			{Method: "POST", Path: "/v1/studio/sdk/resources.delete_skill"},
		},
	})
	if err != nil {
		return err
	}
	aclDeleteRoute := document.Paths["/v1/studio/sdk/acl.delete"]
	if aclDeleteRoute == nil || aclDeleteRoute.Post == nil {
		return fmt.Errorf("native acl.delete OpenAPI route is missing")
	}
	aclDeletedDescription := "ACL grant deleted"
	aclDeleteRoute.Post.Responses = openapi3.Responses{"204": {Description: &aclDeletedDescription}}
	predicateDeleteRoute := document.Paths["/v1/studio/sdk/authorization_predicates.delete"]
	if predicateDeleteRoute == nil || predicateDeleteRoute.Post == nil {
		return fmt.Errorf("native authorization_predicates.delete OpenAPI route is missing")
	}
	predicateDeletedDescription := "Authorization predicate deleted"
	predicateDeleteRoute.Post.Responses = openapi3.Responses{"204": {Description: &predicateDeletedDescription}}
	connectorDeleteRoute := document.Paths["/v1/studio/sdk/connectors.delete"]
	if connectorDeleteRoute == nil || connectorDeleteRoute.Post == nil {
		return fmt.Errorf("native connectors.delete OpenAPI route is missing")
	}
	connectorDeletedDescription := "Connector archived"
	connectorDeleteRoute.Post.Responses = openapi3.Responses{"204": {Description: &connectorDeletedDescription}}
	// The linked delete handler sets HTTP 204 after its internal soft-delete
	// writer succeeds. Datly's generic custom-handler schema defaults to 200;
	// keep the generated client contract aligned with this no-body SDK route.
	deleteRoute := document.Paths["/v1/studio/sdk/namespaces.delete"]
	if deleteRoute == nil || deleteRoute.Post == nil {
		return fmt.Errorf("native namespaces.delete OpenAPI route is missing")
	}
	deletedDescription := "Namespace archived"
	deleteRoute.Post.Responses = openapi3.Responses{"204": {Description: &deletedDescription}}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err = os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	return os.WriteFile(output, data, 0o644)
}

func compile(holder, inputType, outputType reflect.Type, resources *resource.Store,
	types *typecatalog.Catalog, codec xcodec.Factory, handlers ...rhandler.TypedHandler) (*registry.RegisteredComponent, error) {
	field, ok := holder.FieldByName("Contract")
	if !ok {
		return nil, fmt.Errorf("component %s contract is missing", holder)
	}
	metadata, present, err := tag.ParseComponent(field.Tag)
	if err != nil || !present {
		return nil, fmt.Errorf("parse %s metadata: %w", holder, err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
		PackageName: "reader", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(inputType, outputType)
	if err != nil {
		return nil, err
	}
	var handler rhandler.TypedHandler
	if len(handlers) > 0 {
		handler = handlers[0]
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: inputType, OutputType: outputType, Resources: resources, Types: types, CodecFactory: codec,
		Handler: handler, HandlerOwnedOutput: handler != nil})
	if err != nil {
		return nil, err
	}
	return &registry.RegisteredComponent{Component: artifact.Component, Input: artifact.Input,
		Output: artifact.Output, OutputType: outputType}, nil
}

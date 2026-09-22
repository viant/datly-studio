package writer

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for config.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Configs []*ReportCubeConfig `parameter:"Configs,kind=body,in=data,dataType=[]*ReportCubeConfig" view:"config,type=ReportCubeConfig,entityHooks=CubeRules,table=report_cube_configs" sql:"uri=studio_report_cube_configs_writer_config:sql/config.sql"`
	ConfigKeys []ConfigKeysRow `parameter:"ConfigKeys,kind=param,in=Configs,cardinality=Many" codec:"structql,'uri=studio_report_cube_configs_writer_config:sql/config_keys.sql'"`
	CurrentConfig []*CurrentConfigView `parameter:"CurrentConfig,kind=view,in=CurrentConfig,cardinality=Many" view:"CurrentConfig,table=report_cube_configs" sql:"uri=studio_report_cube_configs_writer_config:sql/current_config.sql"`
	_configHandlerReadIndexes *ConfigHandlerReadIndexes `json:"-" sqlx:"-"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	Configs bool
	ConfigKeys bool
	CurrentConfig bool
}

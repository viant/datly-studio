package writer

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for generation.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Generations []*RuntimeGeneration `parameter:"Generations,kind=body,in=data,dataType=[]*RuntimeGeneration" view:"generation,type=RuntimeGeneration,entityHooks=GenerationRules,table=runtime_generations" sql:"uri=studio_runtime_generations_writer_generation:sql/generation.sql"`
	GenerationKeys []GenerationKeysRow `parameter:"GenerationKeys,kind=param,in=Generations,cardinality=Many" codec:"structql,'uri=studio_runtime_generations_writer_generation:sql/generation_keys.sql'"`
	CurrentGeneration []*CurrentGenerationView `parameter:"CurrentGeneration,kind=view,in=CurrentGeneration,cardinality=Many" view:"CurrentGeneration,table=runtime_generations" sql:"uri=studio_runtime_generations_writer_generation:sql/current_generation.sql"`
	_generationHandlerReadIndexes *GenerationHandlerReadIndexes `json:"-" sqlx:"-"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	Generations bool
	GenerationKeys bool
	CurrentGeneration bool
}

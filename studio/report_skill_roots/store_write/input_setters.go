package store_write

func (input *Input) SetSkills(value []*StoredSkill) {
	if input == nil {
		return
	}
	input.Skills = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Skills = true
}

func (input *Input) SetSkillKeys(value []SkillKeysRow) {
	if input == nil {
		return
	}
	input.SkillKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.SkillKeys = true
}

func (input *Input) SetCurrentSkill(value []*CurrentSkillView) {
	if input == nil {
		return
	}
	input.CurrentSkill = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentSkill = true
}

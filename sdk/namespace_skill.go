package sdk

import "time"

// NamespaceSkillFile is relative to a namespace skill root. Binary content is
// standard base64; text content is UTF-8.
type NamespaceSkillFile struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	IsBinary bool   `json:"isBinary,omitempty"`
}

// NamespaceSkillBundle preserves SKILL.md, including its YAML frontmatter.
type NamespaceSkillBundle struct {
	Title   string               `json:"title,omitempty"`
	Content string               `json:"content"`
	Files   []NamespaceSkillFile `json:"files,omitempty"`
}

type NamespaceSkill struct {
	NamespaceID       string    `json:"namespaceId"`
	SkillID           string    `json:"skillId"`
	Name              string    `json:"name"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	LatestRevision    int       `json:"latestRevision"`
	PublishedRevision int       `json:"publishedRevision"`
	Status            string    `json:"status"`
	ETag              int64     `json:"etag"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type NamespaceSkillRevision struct {
	NamespaceID   string               `json:"namespaceId"`
	SkillID       string               `json:"skillId"`
	Revision      int                  `json:"revision"`
	ContentSHA256 string               `json:"contentSha256"`
	Bundle        NamespaceSkillBundle `json:"bundle"`
	CreatedAt     time.Time            `json:"createdAt"`
	CreatedBy     string               `json:"createdBy"`
}

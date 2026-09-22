package sdk

import "context"

const (
	OperationACLList   = "acl.list"
	OperationACLUpsert = "acl.upsert"
	OperationACLDelete = "acl.delete"
)

type ReportACL struct {
	ReportID    string `json:"reportId"`
	SubjectType string `json:"subjectType"`
	SubjectID   string `json:"subjectId"`
	CanView     bool   `json:"canView"`
	CanRun      bool   `json:"canRun"`
	CanEdit     bool   `json:"canEdit"`
	CanPublish  bool   `json:"canPublish"`
	CanUseDQL   bool   `json:"canUseDql"`
	ETag        int64  `json:"etag,omitempty"`
}

type ACLService interface {
	List(context.Context, string) ([]*ReportACL, error)
	Upsert(context.Context, ReportACL) (*ReportACL, error)
	Delete(context.Context, string, string, string, int64) error
}

type aclClient struct{ transport Transport }

func (c aclClient) List(ctx context.Context, reportID string) ([]*ReportACL, error) {
	result := struct {
		Items []*ReportACL `json:"items"`
	}{}
	if err := c.transport.Invoke(ctx, OperationACLList, map[string]string{"reportId": reportID}, &result); err != nil {
		return nil, err
	}
	return result.Items, nil
}
func (c aclClient) Upsert(ctx context.Context, value ReportACL) (*ReportACL, error) {
	return invoke[ReportACL](ctx, c.transport, OperationACLUpsert, value)
}
func (c aclClient) Delete(ctx context.Context, reportID, subjectType, subjectID string, etag int64) error {
	return invokeEmpty(ctx, c.transport, OperationACLDelete, struct {
		ReportID    string `json:"reportId"`
		SubjectType string `json:"subjectType"`
		SubjectID   string `json:"subjectId"`
		ETag        int64  `json:"etag"`
	}{reportID, subjectType, subjectID, etag})
}

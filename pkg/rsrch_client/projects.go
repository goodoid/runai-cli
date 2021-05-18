package rsrch_client

import (
	"context"
	"fmt"
	rsrch_api "github.com/run-ai/researcher-service/server/pkg/runai/api"
	"net/http"
)

type ProjectListOptions struct {
    IncludeDeleted bool
}

type ProjectListResponse []rsrch_api.Project

func (c *RsrchClient) ProjectList(ctx context.Context, options *ProjectListOptions) (*ProjectListResponse, error) {

    url := c.BaseURL + ProjectsURL
    if options != nil {
        if options.IncludeDeleted {
            url = url + "?includeDeleted=true"
        }
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(url), nil)
    if err != nil {
        return nil, err
    }

    res := ProjectListResponse{}
    if _, err := c.sendRequest(req, &res); err != nil {
        return nil, err
    }

    return &res, nil
}


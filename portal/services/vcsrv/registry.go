package vcsrv

/*
registry vcs 实现
*/

import (
	"bytes"
	"cloudiac/portal/consts"
	"cloudiac/portal/consts/e"
	"cloudiac/portal/models"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strings"
)

type RegistryVcs struct {
	vcs      *models.Vcs
	basePath string
}

func newRegistryVcs(vcs *models.Vcs) (VcsIface, error) {

	return &RegistryVcs{vcs: vcs, basePath: consts.RegistryVcsBasePath}, nil
}

// TODO
func (rv *RegistryVcs) GetRepo(repoPath string) (RepoIface, error) {
	return &RegistryRepo{vcs: rv.vcs, basePath: rv.basePath, repoPath: repoPath}, nil
}

// TODO
func (rv *RegistryVcs) ListRepos(namespace string, search string, limit, offset int) ([]RepoIface, int64, error) {

	return nil, 0, nil
}

type RegistryRepo struct {
	vcs      *models.Vcs
	basePath string
	repoPath string // vcs 下repo的相对路径
}

func (r *RegistryRepo) ListBranches() ([]string, error) {
	path := fmt.Sprintf("%s/api/v1/vcs/repo/branches", r.vcs.Address)
	_, body, err := registryVcsRequest(path, "GET", map[string]string{
		"path": r.repoPath,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Result struct {
			Branches []string `json:"branches"`
		} `json:"result"`
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Result.Branches, nil
}

func (r *RegistryRepo) ListTags() ([]string, error) {
	path := fmt.Sprintf("%s/api/v1/vcs/repo/tags", r.vcs.Address)
	_, body, err := registryVcsRequest(path, "GET", map[string]string{
		"path": r.repoPath,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Result struct {
			Tags []string `json:"tags"`
		} `json:"result"`
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Result.Tags, nil
}

func (r *RegistryRepo) BranchCommitId(branch string) (string, error) {
	path := fmt.Sprintf("%s/api/v1/vcs/repo/branch_commit_id", r.vcs.Address)
	_, body, err := registryVcsRequest(path, "GET", map[string]string{
		"path":   r.repoPath,
		"branch": branch,
	})
	if err != nil {
		return "", err
	}

	var resp struct {
		Result struct {
			CommitId string `json:"commitId"`
		} `json:"result"`
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", err
	}

	return resp.Result.CommitId, nil
}

func (r *RegistryRepo) ListFiles(opt VcsIfaceOptions) ([]string, error) {
	path := fmt.Sprintf("%s/api/v1/vcs/repo/files", r.vcs.Address)
	recursive := "false"
	if opt.Recursive {
		recursive = "true"
	}
	_, body, err := registryVcsRequest(path, "GET", map[string]string{
		"path":      r.repoPath,
		"ref":       opt.Ref,
		"filePath":  opt.Path,
		"search":    opt.Search,
		"recursive": recursive,
		"limit":     fmt.Sprintf("%d", opt.Limit),
		"offset":    fmt.Sprintf("%d", opt.Offset),
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Result struct {
			Files []string `json:"files"`
		} `json:"result"`
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Result.Files, nil
}

func (r *RegistryRepo) ReadFileContent(revision string, filePath string) (content []byte, err error) {
	path := fmt.Sprintf("%s/api/v1/vcs/repo/file_content", r.vcs.Address)
	_, body, err := registryVcsRequest(path, "GET", map[string]string{
		"path":     r.repoPath,
		"branch":   revision,
		"filePath": filePath,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code          int    `json:"code"`
		Message       string `json:"message"`
		MessageDetail string `json:"messageDetail"`
		Result        struct {
			Content string `json:"content"`
		} `json:"result"`
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Code != 0 && strings.Contains(resp.MessageDetail, "file not exists") {
		return nil, e.New(e.ObjectNotExists)
	}

	return []byte(resp.Result.Content), nil
}

func (r *RegistryRepo) FormatRepoSearch() (*Projects, e.Error) {
	path := fmt.Sprintf("%s/api/v1/vcs/repo/info", r.vcs.Address)
	_, body, err := registryVcsRequest(path, "GET", map[string]string{
		"path": r.repoPath,
	})
	if err != nil {
		return nil, e.New(e.InternalError, err)
	}

	var resp struct {
		Result *Projects `json:"result"`
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, e.New(e.InternalError, err)
	}

	return resp.Result, nil
}

func (r *RegistryRepo) DefaultBranch() string {
	path := fmt.Sprintf("%s/api/v1/vcs/repo/default_branch", r.vcs.Address)
	_, body, err := registryVcsRequest(path, "GET", map[string]string{
		"path": r.repoPath,
	})
	if err != nil {
		return ""
	}

	var resp struct {
		Result struct {
			DefaultBranch string `json:"defaultBranch"`
		} `json:"result"`
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return ""
	}

	return resp.Result.DefaultBranch
}

func (r *RegistryRepo) AddWebhook(url string) error {
	return nil
}

func (r *RegistryRepo) ListWebhook() ([]ProjectsHook, error) {
	ph := make([]ProjectsHook, 0)
	return ph, nil
}

func (r *RegistryRepo) DeleteWebhook(id int) error {
	return nil
}

func (r *RegistryRepo) CreatePrComment(prId int, comment string) error {

	return nil
}

func registryVcsRequest(path, method string, params map[string]string) (*http.Response, []byte, error) {
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	for k, v := range params {
		_ = writer.WriteField(k, v)
	}
	err := writer.Close()
	if err != nil {
		return nil, nil, err
	}

	client := &http.Client{}
	req, err := http.NewRequest(method, path, payload)

	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	return res, body, err
}

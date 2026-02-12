package signal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type APIClient struct {
	baseURL string
	number  string
	client  *http.Client
}

func NewAPIClient(baseURL, number string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		number:  number,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type GroupDetail struct {
	ID            string   `json:"id"`
	InternalID    string   `json:"internal_id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Members       []string `json:"members"`
	Admins        []string `json:"admins"`
	Blocked       bool     `json:"blocked"`
	InviteLink    string   `json:"invite_link"`
	PermissionStr string   `json:"permission_add_member"`
}

type ContactDetail struct {
	Number      string `json:"number"`
	UUID        string `json:"uuid"`
	ProfileName string `json:"name"`
	Color       string `json:"color"`
	Blocked     bool   `json:"blocked"`
}

func (a *APIClient) ListGroups() ([]GroupDetail, error) {
	url := fmt.Sprintf("%s/v1/groups/%s", a.baseURL, a.number)
	resp, err := a.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list groups: status %d: %s", resp.StatusCode, body)
	}

	var groups []GroupDetail
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return nil, fmt.Errorf("list groups decode: %w", err)
	}
	return groups, nil
}

func (a *APIClient) GetGroup(groupID string) (*GroupDetail, error) {
	url := fmt.Sprintf("%s/v1/groups/%s/%s", a.baseURL, a.number, groupID)
	resp, err := a.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get group: status %d: %s", resp.StatusCode, body)
	}

	var group GroupDetail
	if err := json.NewDecoder(resp.Body).Decode(&group); err != nil {
		return nil, fmt.Errorf("get group decode: %w", err)
	}
	return &group, nil
}

func (a *APIClient) ListContacts() ([]ContactDetail, error) {
	url := fmt.Sprintf("%s/v1/contacts/%s", a.baseURL, a.number)
	resp, err := a.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("list contacts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list contacts: status %d: %s", resp.StatusCode, body)
	}

	var contacts []ContactDetail
	if err := json.NewDecoder(resp.Body).Decode(&contacts); err != nil {
		return nil, fmt.Errorf("list contacts decode: %w", err)
	}
	return contacts, nil
}

func (a *APIClient) DownloadAttachment(attachmentID string) (io.ReadCloser, string, error) {
	url := fmt.Sprintf("%s/v1/attachments/%s", a.baseURL, attachmentID)
	resp, err := a.client.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("download attachment: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", fmt.Errorf("download attachment: status %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	return resp.Body, contentType, nil
}

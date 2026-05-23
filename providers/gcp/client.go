package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/squeak-the-cloud/squeak/output"
	"google.golang.org/api/iam/v1"
)

type GcpRoleAuditResult struct {
	RoleName                 string   `json:"role_name"`
	Title                    string   `json:"title"`
	Description              string   `json:"description"`
	Stage                    string   `json:"stage"`
	IncludedPermissions      []string `json:"included_permissions"`
	WildcardPermissions      []string `json:"wildcard_permissions"`
	HasFullAdmin             bool     `json:"has_full_admin"`
	NonAuditableDetails      []string `json:"non_auditable_details"`
	PrivilegeEscalationPaths []string `json:"privilege_escalation_paths"`
}

type GcpKey struct {
	KeyID        string `json:"key_id"`
	KeyType      string `json:"key_type"`
	ValidBefore  string `json:"valid_before"`
	ValidAfter   string `json:"valid_after"`
	IsUserManaged bool   `json:"is_user_managed"`
	AgeDays      int    `json:"age_days"`
	IsOldKey     bool   `json:"is_old_key"` // keys > 90 days
}

type GcpServiceAccountAuditResult struct {
	Name                string   `json:"name"`
	Email               string   `json:"email"`
	DisplayName         string   `json:"display_name"`
	Disabled            bool     `json:"disabled"`
	Keys                []GcpKey `json:"keys"`
	NonAuditableDetails []string `json:"non_auditable_details"`
	HasActiveUserKeys   bool     `json:"has_active_user_keys"`
}

func Run() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	output.LogInfo("Initializing GCP IAM client...")
	iamService, err := iam.NewService(ctx)
	if err != nil {
		output.LogCritical(fmt.Sprintf("failed to initialize GCP IAM service: %v", err))
		return
	}

	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		output.LogInfo("GCP_PROJECT_ID env var not set. Attempting dynamic discovery from credentials JSON...")
		credPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		if credPath != "" {
			if data, err := os.ReadFile(credPath); err == nil {
				var creds map[string]interface{}
				if err := json.Unmarshal(data, &creds); err == nil {
					if pid, ok := creds["project_id"].(string); ok && pid != "" {
						projectID = pid
						output.LogSuccess(fmt.Sprintf("Discovered GCP Project ID dynamically: %s", projectID))
					}
				}
			}
		}
	}

	if projectID == "" {
		output.LogWarning("Dynamic project discovery failed or returned empty. Defaulting to dummy project ID.")
		projectID = "dummy-project-id"
	}

	parent := fmt.Sprintf("projects/%s", projectID)

	// 1. Audit Service Accounts and their Keys (with PageToken Loop)
	output.LogInfo("Auditing GCP Service Accounts...")
	var accounts []*iam.ServiceAccount
	pageToken := ""

	for {
		call := iamService.Projects.ServiceAccounts.List(parent).Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		response, err := call.Do()
		if err != nil {
			output.LogCritical(fmt.Sprintf("failed to list GCP service accounts: %v", err))
			break
		}

		accounts = append(accounts, response.Accounts...)
		if response.NextPageToken == "" {
			break
		}
		pageToken = response.NextPageToken
	}

	for index, sa := range accounts {
		output.LogInfo(fmt.Sprintf("Auditing Service Account: %s", sa.Email))

		result := GcpServiceAccountAuditResult{
			Name:                sa.Name,
			Email:               sa.Email,
			DisplayName:         sa.DisplayName,
			Disabled:            sa.Disabled,
			Keys:                []GcpKey{},
			NonAuditableDetails: []string{},
			HasActiveUserKeys:   false,
		}

		// List SA Keys
		keysResponse, err := iamService.Projects.ServiceAccounts.Keys.List(sa.Name).Context(ctx).Do()
		if err == nil {
			for _, key := range keysResponse.Keys {
				isUserManaged := key.KeyType == "USER_MANAGED"

				// Parse key creation date to calculate key age
				var ageDays int
				var isOldKey bool
				var validAfterTime time.Time

				// Google API returns RFC3339 timestamps for key validity/creation
				if key.ValidAfterTime != "" {
					parsedTime, parseErr := time.Parse(time.RFC3339, key.ValidAfterTime)
					if parseErr == nil {
						validAfterTime = parsedTime
						age := time.Since(validAfterTime)
						ageDays = int(age.Hours() / 24)
						if ageDays > 90 {
							isOldKey = true
						}
					}
				}

				if isUserManaged {
					result.HasActiveUserKeys = true
				}

				result.Keys = append(result.Keys, GcpKey{
					KeyID:         key.Name,
					KeyType:       key.KeyType,
					ValidBefore:   key.ValidBeforeTime,
					ValidAfter:    key.ValidAfterTime,
					IsUserManaged: isUserManaged,
					AgeDays:       ageDays,
					IsOldKey:      isOldKey,
				})
			}
		} else {
			result.NonAuditableDetails = append(result.NonAuditableDetails, fmt.Sprintf("failed to list keys: %v", err))
		}

		filename := fmt.Sprintf("gcp_service_account_%d.json", index+1)
		if writeErr := output.WriteResult(filename, result); writeErr != nil {
			output.LogCritical(fmt.Sprintf("failed to write audit result for SA %s: %v", sa.Email, writeErr))
		} else {
			output.LogSuccess(fmt.Sprintf("GCP service account %s audited", sa.Email))
		}
	}

	// 2. Audit Custom IAM Roles (with PageToken Loop)
	output.LogInfo("Auditing GCP Custom Roles...")
	var customRoles []*iam.Role
	rolePageToken := ""

	for {
		call := iamService.Projects.Roles.List(parent).Context(ctx)
		if rolePageToken != "" {
			call = call.PageToken(rolePageToken)
		}

		response, err := call.Do()
		if err != nil {
			output.LogCritical(fmt.Sprintf("failed to list GCP project roles: %v", err))
			break
		}

		customRoles = append(customRoles, response.Roles...)
		if response.NextPageToken == "" {
			break
		}
		rolePageToken = response.NextPageToken
	}

	// Escalation APIs mapping
	escalationPermissions := map[string]string{
		"iam.serviceaccounts.actas":          "Impersonate Service Account (Allows user to act as the service account)",
		"iam.serviceaccounts.getaccesstoken": "Get Service Account Access Token (Allows generating temporary OAuth2 tokens)",
		"iam.serviceaccounts.signblob":       "Sign Blob (Allows signing payloads as the service account)",
		"iam.serviceaccounts.signjwt":        "Sign JWT (Allows signing JWTs as the service account)",
		"iam.serviceaccounts.setiampolicy":   "Set Service Account IAM Policy (Allows granting oneself permissions on the service account)",
		"resourcemanager.projects.setiampolicy": "Set Project IAM Policy (Allows granting oneself permissions on the whole project)",
		"compute.instances.create":           "Create VM Instances (Allows deploying new instances with privileged service accounts attached)",
		"deploymentmanager.deployments.create": "Create Deployments (Allows deployment of resource templates with administrator scope)",
	}

	for index, role := range customRoles {
		output.LogInfo(fmt.Sprintf("Auditing Custom Role: %s", role.Title))

		result := GcpRoleAuditResult{
			RoleName:                 role.Name,
			Title:                    role.Title,
			Description:              role.Description,
			Stage:                    role.Stage,
			IncludedPermissions:      []string{},
			WildcardPermissions:      []string{},
			HasFullAdmin:             false,
			NonAuditableDetails:      []string{},
			PrivilegeEscalationPaths: []string{},
		}

		// Get full role details (including permissions list)
		fullRole, err := iamService.Projects.Roles.Get(role.Name).Context(ctx).Do()
		if err == nil {
			for _, perm := range fullRole.IncludedPermissions {
				result.IncludedPermissions = append(result.IncludedPermissions, perm)
				lowerPerm := strings.ToLower(perm)

				if perm == "*" {
					result.HasFullAdmin = true
					result.WildcardPermissions = append(result.WildcardPermissions, perm)
					// Full wildcard includes all escalations
					for _, pathDesc := range escalationPermissions {
						result.PrivilegeEscalationPaths = append(result.PrivilegeEscalationPaths, pathDesc)
					}
				} else if strings.Contains(perm, "*") {
					result.WildcardPermissions = append(result.WildcardPermissions, perm)
					// Wildcard matching prefix
					cleanPrefix := strings.ReplaceAll(lowerPerm, "*", "")
					for key, pathDesc := range escalationPermissions {
						if strings.HasPrefix(key, cleanPrefix) {
							result.PrivilegeEscalationPaths = append(result.PrivilegeEscalationPaths, pathDesc)
						}
					}
				} else {
					// Exact match
					if pathDesc, found := escalationPermissions[lowerPerm]; found {
						result.PrivilegeEscalationPaths = append(result.PrivilegeEscalationPaths, pathDesc)
					}
				}
			}
		} else {
			result.NonAuditableDetails = append(result.NonAuditableDetails, fmt.Sprintf("failed to fetch role details: %v", err))
		}

		filename := fmt.Sprintf("gcp_role_%d.json", index+1)
		if writeErr := output.WriteResult(filename, result); writeErr != nil {
			output.LogCritical(fmt.Sprintf("failed to write audit result for GCP role %s: %v", role.Title, writeErr))
		} else {
			output.LogSuccess(fmt.Sprintf("GCP role %s audited", role.Title))
		}
	}
}

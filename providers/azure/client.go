package azure

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/squeak-the-cloud/squeak/output"
)

type RoleDefinitionAuditResult struct {
	RoleName                 string   `json:"role_name"`
	RoleID                   string   `json:"role_id"`
	RoleType                 string   `json:"role_type"`
	Description              string   `json:"description"`
	Actions                  []string `json:"actions"`
	NotActions               []string `json:"not_actions"`
	DataActions              []string `json:"data_actions"`
	NotDataActions           []string `json:"not_data_actions"`
	HasFullAdmin             bool     `json:"has_full_admin"`
	WildcardActions          []string `json:"wildcard_actions"`
	NonAuditablePermissions []string `json:"non_auditable_permissions"`
	PrivilegeEscalationPaths []string `json:"privilege_escalation_paths"`
}

type RoleAssignmentAuditResult struct {
	AssignmentID     string `json:"assignment_id"`
	Scope            string `json:"scope"`
	RoleDefinitionID string `json:"role_definition_id"`
	RoleName         string `json:"role_name,omitempty"`
	PrincipalID      string `json:"principal_id"`
	PrincipalType    string `json:"principal_type,omitempty"`
}

func Run() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	output.LogInfo("Initializing Azure identity client...")
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		output.LogCritical(fmt.Sprintf("failed to initialize Azure credentials: %v", err))
		return
	}

	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		output.LogInfo("AZURE_SUBSCRIPTION_ID env var not set. Discovering subscriptions dynamically...")
		subClient, err := armsubscriptions.NewClient(cred, nil)
		if err == nil {
			pager := subClient.NewListPager(nil)
			if pager.More() {
				page, pageErr := pager.NextPage(ctx)
				if pageErr == nil && len(page.Value) > 0 {
					subscriptionID = *page.Value[0].SubscriptionID
					output.LogSuccess(fmt.Sprintf("Discovered Azure Subscription dynamically: %s (%s)", *page.Value[0].DisplayName, subscriptionID))
				}
			}
		}
	}

	if subscriptionID == "" {
		output.LogWarning("Dynamic subscription discovery failed or returned empty. Defaulting to dummy subscription ID.")
		subscriptionID = "00000000-0000-0000-0000-000000000000"
	}

	scope := fmt.Sprintf("/subscriptions/%s", subscriptionID)

	clientFactory, err := armauthorization.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		output.LogCritical(fmt.Sprintf("failed to create Azure authorization client factory: %v", err))
		return
	}

	definitionsClient := clientFactory.NewRoleDefinitionsClient()
	output.LogInfo("Listing and auditing Azure Role Definitions...")

	pager := definitionsClient.NewListPager(scope, &armauthorization.RoleDefinitionsClientListOptions{})
	index := 0

	// Track role definition ID to name mappings for assignments resolution
	roleDefIDToName := make(map[string]string)

	// Escalation APIs list
	escalationActions := map[string]string{
		"microsoft.authorization/roleassignments/write":                  "Assign Role (Allows assignment of any role, leading to full takeover)",
		"microsoft.authorization/roledefinitions/write":                  "Modify Role (Allows editing roles to grant self additional rights)",
		"microsoft.compute/virtualmachines/runcommand/action":            "Virtual Machine RunCommand (Allows running root scripts on virtual instances)",
		"microsoft.compute/virtualmachines/write":                        "Modify VM (Allows attaching managed identity with higher privileges)",
		"microsoft.resources/deployments/write":                          "Template Deployments (Allows provisioning and executing privileged resource templates)",
		"microsoft.automation/automationaccounts/runbooks/write":          "Automation Runbooks (Allows executing arbitrary administration code)",
		"microsoft.managedidentity/userassignedidentities/assign/action": "Assign Managed Identity (Allows attaching identities to virtual resources)",
	}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			output.LogCritical(fmt.Sprintf("failed to get page of role definitions: %v", err))
			break
		}

		for _, roleDef := range page.Value {
			if roleDef.Properties == nil {
				continue
			}

			roleName := ""
			if roleDef.Properties.RoleName != nil {
				roleName = *roleDef.Properties.RoleName
			}

			roleID := ""
			if roleDef.Name != nil {
				roleID = *roleDef.Name
			}

			// Save to mapping dictionary (both simple ID and full path ID if possible)
			if roleID != "" && roleName != "" {
				roleDefIDToName[strings.ToLower(roleID)] = roleName
				if roleDef.ID != nil {
					roleDefIDToName[strings.ToLower(*roleDef.ID)] = roleName
				}
			}

			roleType := ""
			if roleDef.Properties.RoleType != nil {
				roleType = *roleDef.Properties.RoleType
			}

			description := ""
			if roleDef.Properties.Description != nil {
				description = *roleDef.Properties.Description
			}

			result := RoleDefinitionAuditResult{
				RoleName:                 roleName,
				RoleID:                   roleID,
				RoleType:                 roleType,
				Description:              description,
				Actions:                  []string{},
				NotActions:               []string{},
				DataActions:              []string{},
				NotDataActions:           []string{},
				HasFullAdmin:             false,
				WildcardActions:          []string{},
				NonAuditablePermissions: []string{},
				PrivilegeEscalationPaths: []string{},
			}

			for _, perm := range roleDef.Properties.Permissions {
				if perm == nil {
					continue
				}

				for _, act := range perm.Actions {
					if act != nil {
						actionStr := *act
						result.Actions = append(result.Actions, actionStr)
						lowerAct := strings.ToLower(actionStr)

						if actionStr == "*" {
							result.HasFullAdmin = true
							result.WildcardActions = append(result.WildcardActions, actionStr)
							// Wildcard allows all escalations
							for _, pathDesc := range escalationActions {
								result.PrivilegeEscalationPaths = append(result.PrivilegeEscalationPaths, pathDesc)
							}
						} else if strings.Contains(actionStr, "*") {
							result.WildcardActions = append(result.WildcardActions, actionStr)
							// Wildcard matching prefix
							cleanPrefix := strings.ReplaceAll(lowerAct, "*", "")
							for key, pathDesc := range escalationActions {
								if strings.HasPrefix(key, cleanPrefix) {
									result.PrivilegeEscalationPaths = append(result.PrivilegeEscalationPaths, pathDesc)
								}
							}
						} else {
							// Exact matching
							if pathDesc, found := escalationActions[lowerAct]; found {
								result.PrivilegeEscalationPaths = append(result.PrivilegeEscalationPaths, pathDesc)
							}
						}
					}
				}

				for _, notAct := range perm.NotActions {
					if notAct != nil {
						result.NotActions = append(result.NotActions, *notAct)
					}
				}

				for _, dAct := range perm.DataActions {
					if dAct != nil {
						result.DataActions = append(result.DataActions, *dAct)
					}
				}

				for _, notDAct := range perm.NotDataActions {
					if notDAct != nil {
						result.NotDataActions = append(result.NotDataActions, *notDAct)
					}
				}
			}

			index++
			filename := fmt.Sprintf("azure_role_%d.json", index)
			if writeErr := output.WriteResult(filename, result); writeErr != nil {
				output.LogCritical(fmt.Sprintf("failed to write audit result for Azure role %s: %v", roleName, writeErr))
			} else {
				output.LogSuccess(fmt.Sprintf("Azure role %s audited", roleName))
			}
		}
	}

	// 2. Audit Role Assignments (New capability)
	output.LogInfo("Listing and auditing Azure Role Assignments...")
	assignmentsClient := clientFactory.NewRoleAssignmentsClient()
	assignPager := assignmentsClient.NewListForSubscriptionPager(nil)
	assignIndex := 0

	for assignPager.More() {
		page, err := assignPager.NextPage(ctx)
		if err != nil {
			output.LogCritical(fmt.Sprintf("failed to list role assignments: %v", err))
			break
		}

		for _, assignment := range page.Value {
			if assignment.Properties == nil {
				continue
			}

			assignID := ""
			if assignment.Name != nil {
				assignID = *assignment.Name
			}

			roleDefID := ""
			if assignment.Properties.RoleDefinitionID != nil {
				roleDefID = *assignment.Properties.RoleDefinitionID
			}

			principalID := ""
			if assignment.Properties.PrincipalID != nil {
				principalID = *assignment.Properties.PrincipalID
			}

			principalType := ""
			if assignment.Properties.PrincipalType != nil {
				principalType = string(*assignment.Properties.PrincipalType)
			}

			assignmentScope := ""
			if assignment.Properties.Scope != nil {
				assignmentScope = *assignment.Properties.Scope
			}

			// Attempt to resolve friendly role name
			resolvedName := ""
			if roleDefID != "" {
				resolvedName = roleDefIDToName[strings.ToLower(roleDefID)]
				if resolvedName == "" {
					// Extract role def guid from path
					parts := strings.Split(roleDefID, "/")
					if len(parts) > 0 {
						guid := parts[len(parts)-1]
						resolvedName = roleDefIDToName[strings.ToLower(guid)]
					}
				}
			}

			result := RoleAssignmentAuditResult{
				AssignmentID:     assignID,
				Scope:            assignmentScope,
				RoleDefinitionID: roleDefID,
				RoleName:         resolvedName,
				PrincipalID:      principalID,
				PrincipalType:    principalType,
			}

			assignIndex++
			filename := fmt.Sprintf("azure_role_assignment_%d.json", assignIndex)
			if writeErr := output.WriteResult(filename, result); writeErr != nil {
				output.LogCritical(fmt.Sprintf("failed to write Azure role assignment audit: %v", writeErr))
			}
		}
	}
}
